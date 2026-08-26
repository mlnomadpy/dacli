// Issue #746: binary/version and sandbox-flag checks can pass while the exact
// provider startup transport is forbidden by the effective OS sandbox. This
// file is the adapter seam: provider argv and prose classification stay here;
// orchestration consumes only store.RuntimeLaunchPreflight.
package execution

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const (
	launchPreflightMaxAge = 5 * time.Minute
)

// A cold Codex process may spend several seconds loading local context before
// it opens the provider turn. The deadline bounds transport readiness, not a
// complete inference; V2 stops as soon as Codex emits turn.started.
var launchPreflightTimeout = 30 * time.Second

type cappedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	remaining int
}

func (w *cappedBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(p)
	if len(p) > w.remaining {
		p = p[:w.remaining]
	}
	_, _ = w.buf.Write(p)
	w.remaining -= len(p)
	return n, nil
}

func (w *cappedBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func hasBehavioralPreflight(rt store.Runtime) bool {
	switch rt.BehavioralPreflight {
	case store.BehavioralPreflightCodexExecJSONV1, store.BehavioralPreflightCodexExecJSONV2, store.BehavioralPreflightClaudePrintV1:
		return true
	default:
		return false
	}
}

type readinessResult struct {
	ready bool
	err   error
}

// scanCodexReadiness consumes only stdout because Codex reserves that stream
// for JSONL. Stderr may contain warnings and is captured independently for
// failure classification. Scanner reassembles fragmented writes into lines;
// the explicit limit prevents an untrusted runtime from growing memory without
// bound before producing a newline.
func scanCodexReadiness(r io.Reader, output io.Writer) readinessResult {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		_, _ = output.Write(append(line, '\n'))
		var event struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(line, &event) == nil && event.Type == "turn.started" {
			return readinessResult{ready: true}
		}
	}
	return readinessResult{err: scanner.Err()}
}

// scanClaudeReadiness recognizes Claude Code's first stream-json event. The
// probe stops before a model turn, while an unauthenticated CLI exits with its
// actionable /login remedy for the adapter classifier below.
func scanClaudeReadiness(r io.Reader, output io.Writer) readinessResult {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		_, _ = output.Write(append(line, '\n'))
		var event struct {
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
		}
		if json.Unmarshal(line, &event) == nil && event.Type == "system" && event.Subtype == "init" {
			return readinessResult{ready: true}
		}
	}
	return readinessResult{err: scanner.Err()}
}

func scanBehavioralReadiness(rt store.Runtime, r io.Reader, output io.Writer) readinessResult {
	if rt.BehavioralPreflight == store.BehavioralPreflightClaudePrintV1 {
		return scanClaudeReadiness(r, output)
	}
	return scanCodexReadiness(r, output)
}

func runBehavioralPreflight(ctx *clikit.Ctx, rt store.Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig bool) store.RuntimeLaunchPreflight {
	started := time.Now().UTC()
	if !hasBehavioralPreflight(rt) {
		return store.RuntimeLaunchPreflight{State: store.LaunchUnsupported, Provenance: store.ProvenanceDeclared, CommandTimestamp: started,
			Detail: "adapter declares no behavioral launch strategy"}
	}
	args := append([]string{}, rt.GlobalArgs...)
	args = append(args, rt.Args...)
	if grant == model.GrantRO {
		args = append(args, rt.SandboxRO...)
	}
	if selectedModel != "" && rt.ModelFlag != "" {
		args = append(args, rt.ModelFlag, selectedModel)
	}
	if rt.BehavioralPreflight == store.BehavioralPreflightClaudePrintV1 {
		args = append(args, "--output-format", "stream-json", "--verbose")
	}

	probeCtx, cancel := context.WithTimeout(context.Background(), launchPreflightTimeout)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, binaryPath, args...)
	cmd.Dir = ctx.Cwd
	cmd.Env = runtimeAllowlistedEnv(rt)
	if rt.Mode == "stdin" {
		cmd.Stdin = strings.NewReader("Return no content.\n")
	} else {
		if rt.Flag != "" {
			cmd.Args = append(cmd.Args, rt.Flag)
		}
		cmd.Args = append(cmd.Args, "Return no content.")
	}
	output := cappedBuffer{remaining: 64 << 10}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return classifyLaunchFailure(rt, err.Error())
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return classifyLaunchFailure(rt, err.Error())
	}
	setNewProcessGroup(cmd)
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			_ = killProcessGroup(cmd.Process.Pid)
		}
		return nil
	}
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Start(); err != nil {
		return classifyLaunchFailure(rt, err.Error())
	}
	readiness := make(chan readinessResult, 1)
	stderrDone := make(chan struct{})
	go func() { readiness <- scanBehavioralReadiness(rt, stdout, &output) }()
	go func() {
		_, _ = io.Copy(&output, stderr)
		close(stderrDone)
	}()

	result := store.RuntimeLaunchPreflight{State: store.LaunchCompatible, Provenance: store.ProvenanceProbed, CommandTimestamp: started,
		Detail: "exact adapter startup readiness event observed"}
	select {
	case scan := <-readiness:
		if scan.ready {
			_ = killProcessGroup(cmd.Process.Pid)
			_ = cmd.Wait()
			// A shell wrapper can fork between emitting readiness and observing
			// the first group signal. Once the leader is reaped, signal the same
			// still-owned group again so a just-forked pipe holder cannot extend
			// a capability probe to the model-turn lifetime.
			_ = killProcessGroup(cmd.Process.Pid)
			_ = stdout.Close()
			_ = stderr.Close()
			<-stderrDone
			return result
		}
		err = cmd.Wait()
		_ = stderr.Close()
		<-stderrDone
		if probeCtx.Err() == context.DeadlineExceeded {
			result.State, result.Layer, result.Detail = store.LaunchTransient, store.LaunchTransport, "behavioral launch readiness exceeded bounded deadline"
			return result
		}
		provider := behavioralProvider(rt)
		if scan.err != nil {
			_, _ = output.Write([]byte("\ninvalid " + provider + " JSONL stream: " + scan.err.Error()))
		}
		if strings.TrimSpace(output.String()) == "" && err == nil {
			_, _ = output.Write([]byte(provider + " exited before a valid readiness event"))
		}
		return classifyLaunchFailure(rt, output.String())
	case <-probeCtx.Done():
		_ = killProcessGroup(cmd.Process.Pid)
		_ = cmd.Wait()
		_ = killProcessGroup(cmd.Process.Pid)
		_ = stdout.Close()
		_ = stderr.Close()
		<-stderrDone
		result.State, result.Layer, result.Detail = store.LaunchTransient, store.LaunchTransport, "behavioral launch readiness exceeded bounded deadline"
		return result
	}
}

func runtimeAllowlistedEnv(rt store.Runtime) []string {
	var env []string
	for _, name := range rt.Env {
		if deniedEnvPassthrough([]string{name}) != "" {
			continue
		}
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return env
}

func classifyCodexLaunchFailure(output string) store.RuntimeLaunchPreflight {
	return classifyLaunchFailure(store.Runtime{BehavioralPreflight: store.BehavioralPreflightCodexExecJSONV2}, output)
}

func behavioralProvider(rt store.Runtime) string {
	if rt.BehavioralPreflight == store.BehavioralPreflightClaudePrintV1 {
		return "Claude Code"
	}
	return "Codex"
}

func classifyLaunchFailure(rt store.Runtime, output string) store.RuntimeLaunchPreflight {
	lower := strings.ToLower(output)
	provider := behavioralProvider(rt)
	result := store.RuntimeLaunchPreflight{State: store.LaunchTransient, Layer: store.LaunchStartup, Provenance: store.ProvenanceProbed, CommandTimestamp: time.Now().UTC(), Detail: provider + " startup handshake failed"}
	switch {
	case strings.Contains(lower, "app-server") && (strings.Contains(lower, "operation not permitted") || strings.Contains(lower, "permission denied")):
		result.State, result.Layer, result.Detail = store.LaunchIncompatible, store.LaunchSandbox, provider+" app-server initialization is forbidden by the effective sandbox"
	case strings.Contains(lower, "not logged in") || strings.Contains(lower, "authentication") || strings.Contains(lower, "unauthorized"):
		remedy := "authenticate the runtime and retry"
		if rt.BehavioralPreflight == store.BehavioralPreflightClaudePrintV1 {
			remedy = "run `/login` in Claude Code, then retry"
		}
		result.State, result.Layer, result.Detail = store.LaunchIncompatible, store.LaunchAuthentication, provider+" authentication is not ready; "+remedy
	case strings.Contains(lower, "quota") || strings.Contains(lower, "rate limit"):
		result.State, result.Layer, result.Detail = store.LaunchTransient, store.LaunchQuota, provider+" quota is temporarily unavailable"
	case strings.Contains(lower, "connection") || strings.Contains(lower, "transport"):
		result.State, result.Layer, result.Detail = store.LaunchTransient, store.LaunchTransport, provider+" startup transport failed"
	}
	return result
}

func launchCompatibility(ctx *clikit.Ctx, w *workspace.Workspace, rt store.Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig, force bool) (store.RuntimeLaunchPreflight, error) {
	if !hasBehavioralPreflight(rt) {
		return runBehavioralPreflight(ctx, rt, binaryPath, grant, selectedModel, allowUserConfig), nil
	}
	now := time.Now().UTC()
	if !force {
		if cached, ok := store.LoadFreshRuntimeLaunchPreflight(w, rt, binaryPath, grant, selectedModel, allowUserConfig, now, launchPreflightMaxAge); ok {
			// Only positive evidence authorizes without executing the exact
			// handshake. Transient failures must remain immediately retryable,
			// and deterministic failures are cheap enough to re-confirm before
			// returning the policy answer.
			if cached.State == store.LaunchCompatible {
				return cached, nil
			}
		}
	}
	result := runBehavioralPreflight(ctx, rt, binaryPath, grant, selectedModel, allowUserConfig)
	if err := store.SaveRuntimeLaunchPreflight(w, rt, binaryPath, grant, selectedModel, allowUserConfig, result); err != nil {
		return result, fmt.Errorf("cache behavioral preflight for runtime %s: %w", rt.Name, err)
	}
	return result, nil
}

func requireLaunchCompatibility(ctx *clikit.Ctx, w *workspace.Workspace, rt store.Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig bool) error {
	result, err := launchCompatibility(ctx, w, rt, binaryPath, grant, selectedModel, allowUserConfig, false)
	if err != nil {
		return err
	}
	return launchResultError(rt, grant, result)
}

func launchResultError(rt store.Runtime, grant model.Grant, result store.RuntimeLaunchPreflight) error {
	switch result.State {
	case store.LaunchCompatible, store.LaunchUnsupported:
		return nil
	case store.LaunchIncompatible:
		return clikit.Refusedf("runtime %s behavioral preflight is incompatible at layer %s: %s. Run `dacli runtime doctor --runtime %s --grant %s`; choose another configured runtime if it reports compatible", rt.Name, result.Layer, result.Detail, rt.Name, grant)
	default:
		return fmt.Errorf("runtime %s behavioral preflight failed transiently at layer %s: %s", rt.Name, result.Layer, result.Detail)
	}
}
