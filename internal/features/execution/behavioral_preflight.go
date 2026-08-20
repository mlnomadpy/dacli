// Issue #746: binary/version and sandbox-flag checks can pass while the exact
// provider startup transport is forbidden by the effective OS sandbox. This
// file is the adapter seam: provider argv and prose classification stay here;
// orchestration consumes only store.RuntimeLaunchPreflight.
package execution

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const (
	launchPreflightMaxAge = 5 * time.Minute
)

var launchPreflightTimeout = 5 * time.Second

type cappedBuffer struct {
	bytes.Buffer
	remaining int
}

func (w *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if len(p) > w.remaining {
		p = p[:w.remaining]
	}
	_, _ = w.Buffer.Write(p)
	w.remaining -= len(p)
	return n, nil
}

func hasBehavioralPreflight(rt store.Runtime) bool {
	return rt.BehavioralPreflight == store.BehavioralPreflightCodexExecJSONV1
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
	cmd.Stdout, cmd.Stderr = &output, &output
	err := cmd.Run()
	result := store.RuntimeLaunchPreflight{State: store.LaunchCompatible, Provenance: store.ProvenanceProbed, CommandTimestamp: started,
		Detail: "exact adapter startup handshake completed"}
	if err == nil {
		return result
	}
	if probeCtx.Err() == context.DeadlineExceeded {
		result.State, result.Layer, result.Detail = store.LaunchTransient, store.LaunchTransport, "behavioral launch handshake exceeded bounded deadline"
		return result
	}
	return classifyCodexLaunchFailure(output.String())
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
	lower := strings.ToLower(output)
	result := store.RuntimeLaunchPreflight{State: store.LaunchTransient, Layer: store.LaunchStartup, Provenance: store.ProvenanceProbed, CommandTimestamp: time.Now().UTC(), Detail: "Codex startup handshake failed"}
	switch {
	case strings.Contains(lower, "app-server") && (strings.Contains(lower, "operation not permitted") || strings.Contains(lower, "permission denied")):
		result.State, result.Layer, result.Detail = store.LaunchIncompatible, store.LaunchSandbox, "Codex app-server initialization is forbidden by the effective sandbox"
	case strings.Contains(lower, "not logged in") || strings.Contains(lower, "authentication") || strings.Contains(lower, "unauthorized"):
		result.State, result.Layer, result.Detail = store.LaunchIncompatible, store.LaunchAuthentication, "Codex authentication is not ready"
	case strings.Contains(lower, "quota") || strings.Contains(lower, "rate limit"):
		result.State, result.Layer, result.Detail = store.LaunchTransient, store.LaunchQuota, "Codex quota is temporarily unavailable"
	case strings.Contains(lower, "connection") || strings.Contains(lower, "transport"):
		result.State, result.Layer, result.Detail = store.LaunchTransient, store.LaunchTransport, "Codex startup transport failed"
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
