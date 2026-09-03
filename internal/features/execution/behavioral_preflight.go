// Issue #746: binary/version and sandbox-flag checks can pass while the exact
// provider startup transport is forbidden by the effective OS sandbox. This
// file is the adapter seam: provider argv and prose classification stay here;
// orchestration consumes only store.RuntimeLaunchPreflight.
package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
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

// Issue #803: Claude Code 2.1.198 can emit system/init before it reports that
// the local session is not authenticated. Keep draining both streams for a
// short, bounded settling window after init; init proves transport startup,
// but absence of a deterministic refusal during this window is what proves
// launch compatibility. This is deliberately much shorter than a model turn.
var claudePostInitSettlingTime = time.Second

type cappedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	remaining int
	full      atomic.Bool
}

func (w *cappedBuffer) Write(p []byte) (int, error) {
	if w.full.Load() {
		return len(p), nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(p)
	if w.remaining == 0 {
		w.full.Store(true)
		return n, nil
	}
	if len(p) > w.remaining {
		p = p[:w.remaining]
	}
	_, _ = w.buf.Write(p)
	w.remaining -= len(p)
	if w.remaining == 0 {
		// Stderr must remain drained after diagnostics are capped, but it
		// must not repeatedly contend with stdout readiness for this lock.
		w.full.Store(true)
	}
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

// readinessWriter observes JSONL without exposing an os/exec pipe to the
// caller. Cmd.Wait waits for its internal writer goroutine before returning,
// so a fast provider exit cannot close a pipe while the final authentication
// diagnostic is still being copied. The previous StdoutPipe/StderrPipe design
// made that race visible under -race instrumentation and occasionally
// classified Claude's /login refusal as a generic startup failure.
type readinessWriter struct {
	mu       sync.Mutex
	rt       store.Runtime
	output   *cappedBuffer
	pending  []byte
	ready    bool
	overlong bool
	observed chan struct{}
}

func newReadinessWriter(rt store.Runtime, output *cappedBuffer) *readinessWriter {
	return &readinessWriter{rt: rt, output: output, observed: make(chan struct{})}
}

func (w *readinessWriter) Write(p []byte) (int, error) {
	_, _ = w.output.Write(p)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.overlong {
		return len(p), nil
	}
	w.pending = append(w.pending, p...)
	for {
		newline := bytes.IndexByte(w.pending, '\n')
		if newline < 0 {
			break
		}
		w.observeLine(w.pending[:newline])
		w.pending = w.pending[newline+1:]
	}
	if len(w.pending) > 1<<20 {
		w.pending = nil
		w.overlong = true
	}
	return len(p), nil
}

// finish recognizes a final unterminated JSONL event after Cmd.Wait has
// completed all writer copies.
func (w *readinessWriter) finish() (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.pending) > 0 && !w.overlong {
		w.observeLine(w.pending)
		w.pending = nil
	}
	if w.overlong {
		return w.ready, fmt.Errorf("JSONL event exceeded 1 MiB")
	}
	return w.ready, nil
}

func (w *readinessWriter) observeLine(line []byte) {
	if w.ready {
		return
	}
	var event struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
	}
	if json.Unmarshal(line, &event) != nil {
		return
	}
	ready := event.Type == "turn.started"
	if w.rt.BehavioralPreflight == store.BehavioralPreflightClaudePrintV1 {
		ready = event.Type == "system" && event.Subtype == "init"
	}
	if ready {
		w.ready = true
		close(w.observed)
	}
}

// terminateBehavioralProcess keeps signaling the owned group until Cmd.Wait
// confirms that the leader and all os/exec output-copy goroutines are done.
// A shell may fork between the first signal and leader exit; one-shot killing
// would then wait for WaitDelay while the new child retained an output fd.
func terminateBehavioralProcess(ctx context.Context, cmd *exec.Cmd, processDone <-chan error) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		_ = killProcessGroup(cmd.Process.Pid)
		select {
		case err := <-processDone:
			_ = killProcessGroup(cmd.Process.Pid)
			return err
		case <-ticker.C:
		case <-ctx.Done():
			_ = killProcessGroup(cmd.Process.Pid)
			return <-processDone
		}
	}
}

func runBehavioralPreflight(ctx *clikit.Ctx, rt store.Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig bool) store.RuntimeLaunchPreflight {
	sandbox := []string(nil)
	if grant == model.GrantRO {
		sandbox = rt.SandboxRO
	}
	return runBehavioralPreflightWithSandbox(ctx, rt, binaryPath, grant, selectedModel, allowUserConfig, sandbox)
}

func runBehavioralPreflightWithSandbox(ctx *clikit.Ctx, rt store.Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig bool, sandbox []string) store.RuntimeLaunchPreflight {
	started := time.Now().UTC()
	if !hasBehavioralPreflight(rt) {
		return store.RuntimeLaunchPreflight{State: store.LaunchUnsupported, Provenance: store.ProvenanceDeclared, CommandTimestamp: started,
			Detail: "adapter declares no behavioral launch strategy"}
	}
	args := append([]string{}, rt.GlobalArgs...)
	args = append(args, rt.Args...)
	args = append(args, sandbox...)
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
	stdoutOutput := cappedBuffer{remaining: 64 << 10}
	stderrOutput := cappedBuffer{remaining: 64 << 10}
	readiness := newReadinessWriter(rt, &stdoutOutput)
	cmd.Stdout = readiness
	cmd.Stderr = &stderrOutput
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
	processDone := make(chan error, 1)
	go func() { processDone <- cmd.Wait() }()

	result := store.RuntimeLaunchPreflight{State: store.LaunchCompatible, Provenance: store.ProvenanceProbed, CommandTimestamp: started,
		Detail: "exact adapter startup readiness event observed"}
	select {
	case <-readiness.observed:
		if rt.BehavioralPreflight == store.BehavioralPreflightClaudePrintV1 {
			timer := time.NewTimer(claudePostInitSettlingTime)
			defer timer.Stop()

			var processErr error
			exitedBeforeSettle := false
			select {
			case processErr = <-processDone:
				exitedBeforeSettle = true
			case <-timer.C:
				processErr = terminateBehavioralProcess(probeCtx, cmd, processDone)
			case <-probeCtx.Done():
				_ = terminateBehavioralProcess(probeCtx, cmd, processDone)
				result.State, result.Layer, result.Detail = store.LaunchTransient, store.LaunchTransport, "behavioral launch readiness exceeded bounded deadline"
				return result
			}
			classified := classifyLaunchFailure(rt, stdoutOutput.String()+"\n"+stderrOutput.String())
			// A natural non-zero exit after init is still a failed launch.
			// During the intentional settling-window kill, only a recognized
			// provider diagnostic overrides the observed readiness event.
			if (exitedBeforeSettle && processErr != nil) || classified.Layer != store.LaunchStartup {
				return classified
			}
			return result
		}
		_ = terminateBehavioralProcess(probeCtx, cmd, processDone)
		return result
	case processErr := <-processDone:
		if probeCtx.Err() == context.DeadlineExceeded {
			result.State, result.Layer, result.Detail = store.LaunchTransient, store.LaunchTransport, "behavioral launch readiness exceeded bounded deadline"
			return result
		}
		ready, scanErr := readiness.finish()
		if ready {
			if rt.BehavioralPreflight != store.BehavioralPreflightClaudePrintV1 {
				return result
			}
			classified := classifyLaunchFailure(rt, stdoutOutput.String()+"\n"+stderrOutput.String())
			if processErr != nil || classified.Layer != store.LaunchStartup {
				return classified
			}
			return result
		}
		provider := behavioralProvider(rt)
		if scanErr != nil {
			_, _ = stdoutOutput.Write([]byte("\ninvalid " + provider + " JSONL stream: " + scanErr.Error()))
		}
		output := stdoutOutput.String() + "\n" + stderrOutput.String()
		if strings.TrimSpace(output) == "" && processErr == nil {
			_, _ = stdoutOutput.Write([]byte(provider + " exited before a valid readiness event"))
			output = stdoutOutput.String() + "\n" + stderrOutput.String()
		}
		return classifyLaunchFailure(rt, output)
	case <-probeCtx.Done():
		_ = terminateBehavioralProcess(probeCtx, cmd, processDone)
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
	case strings.Contains(lower, "not logged in") || strings.Contains(lower, "not authenticated") || strings.Contains(lower, "authentication required") || strings.Contains(lower, "authentication failed") || strings.Contains(lower, "authentication_error") || strings.Contains(lower, "unauthorized"):
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

func launchCompatibility(ctx *clikit.Ctx, w *workspace.Workspace, rt store.Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig, force bool, sandbox []string, resultChannel string) (store.RuntimeLaunchPreflight, error) {
	contract, err := store.BuildRuntimeLaunchContract(rt, binaryPath, grant, selectedModel, allowUserConfig, sandbox, resultChannel)
	if err != nil {
		return store.RuntimeLaunchPreflight{}, err
	}
	if !hasBehavioralPreflight(rt) {
		result := runBehavioralPreflightWithSandbox(ctx, rt, binaryPath, grant, selectedModel, allowUserConfig, sandbox)
		result.Contract = contract
		return result, nil
	}
	now := time.Now().UTC()
	if !force {
		if cached, ok := store.LoadFreshRuntimeLaunchPreflightForContract(w, rt, binaryPath, contract, now, launchPreflightMaxAge); ok {
			// Only positive evidence authorizes without executing the exact
			// handshake. Transient failures must remain immediately retryable,
			// and deterministic failures are cheap enough to re-confirm before
			// returning the policy answer.
			if cached.State == store.LaunchCompatible {
				return cached, nil
			}
		}
	}
	result := runBehavioralPreflightWithSandbox(ctx, rt, binaryPath, grant, selectedModel, allowUserConfig, sandbox)
	result.Contract = contract
	if err := store.SaveRuntimeLaunchPreflightForContract(w, rt, binaryPath, contract, result); err != nil {
		return result, fmt.Errorf("cache behavioral preflight for runtime %s: %w", rt.Name, err)
	}
	return result, nil
}

func requireLaunchCompatibility(ctx *clikit.Ctx, w *workspace.Workspace, rt store.Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig bool, sandbox []string, resultChannel, expectedFingerprint string) (store.RuntimeLaunchContract, error) {
	result, err := launchCompatibility(ctx, w, rt, binaryPath, grant, selectedModel, allowUserConfig, false, sandbox, resultChannel)
	if err != nil {
		return store.RuntimeLaunchContract{}, err
	}
	if expectedFingerprint != "" && result.Contract.Fingerprint != expectedFingerprint {
		return result.Contract, clikit.Refusedf("launch contract fingerprint changed after preflight: expected %s, actual %s; rerun preflight for the exact runtime/model/grant/sandbox/result channel", expectedFingerprint, result.Contract.Fingerprint)
	}
	return result.Contract, launchResultError(rt, grant, result)
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
