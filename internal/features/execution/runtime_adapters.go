package execution

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

func cmdRuntimeAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, err := clikit.ParseFlags(args, "flag", "arg", "sandbox-ro-arg", "model-flag", "token-limit-flag", "execution-capability")
	if err != nil {
		return err
	}
	if err := f.Reject("preset", "harness", "binary", "mode", "flag", "arg", "sandbox-ro-arg", "env", "model-flag", "token-limit-flag", "execution-capability", "skills-native-dir", "skills-context-file", "usage-format", "behavioral-preflight"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli runtime add <name> [--preset claude-code|claude-code-rw|codex|codex-rw|gemini|gemini-rw|copilot|copilot-rw|generic-exec] [--harness family] [--binary b] [--mode stdin|arg] [--flag -p] [--arg a]... [--sandbox-ro-arg a]... [--env NAME]... [--model-flag f] [--token-limit-flag f] [--behavioral-preflight codex-exec-json-v1|codex-exec-json-v2|claude-print-v1]\n(--flag/--arg/--sandbox-ro-arg/--model-flag/--token-limit-flag take their value verbatim, even one starting with -, e.g. --model-flag --model)")
	}
	// A runtime names the binary and env that every child in it executes with —
	// defining one is the most privileged write in the system. Without this
	// gate a read-only agent could add a runtime that runs an arbitrary binary
	// and then spawn into it.
	if err := clikit.RequireRW(id, "adding a runtime"); err != nil {
		return err
	}

	rt := store.Runtime{Name: f.Pos[0]}
	if p := f.Get("preset"); p != "" {
		base, ok := presets[p]
		if !ok {
			return clikit.Usagef("unknown preset %q", p)
		}
		base.Name = rt.Name
		rt = base
	}
	if v := f.Get("binary"); v != "" {
		rt.Binary = v
	}
	if v := f.Get("harness"); v != "" {
		rt.Harness = v
	}
	if rt.Harness == "" {
		rt.Harness = rt.Name
	}
	if v := f.Get("mode"); v != "" {
		rt.Mode = v
	}
	if v := f.Get("flag"); v != "" {
		rt.Flag = v
	}
	if v := f.All("arg"); len(v) > 0 {
		rt.Args = v
	}
	if v := f.All("sandbox-ro-arg"); len(v) > 0 {
		rt.SandboxRO = v
	}
	if v := f.All("env"); len(v) > 0 {
		if bad := deniedEnvPassthrough(v); bad != "" {
			return clikit.Refusedf("env passthrough %q is denied: children run under the user's own Claude Code login, never an inherited API key — passing it would bill the operator's API and leak the credential into the child", bad)
		}
		rt.Env = v
	}
	if v := f.Get("model-flag"); v != "" {
		rt.ModelFlag = v
	}
	if v := f.Get("token-limit-flag"); v != "" {
		rt.TokenLimitFlag = v
	}
	for _, value := range f.All("execution-capability") {
		capability := store.ExecutionCapability(value)
		if capability != store.ExecutionCapabilityLocalCoordinationSocket {
			return clikit.Usagef("unsupported --execution-capability %q (supported: %s)", value, store.ExecutionCapabilityLocalCoordinationSocket)
		}
		rt.ExecutionCapabilities = append(rt.ExecutionCapabilities, capability)
	}
	if v := f.Get("skills-native-dir"); v != "" {
		rt.SkillsNativeDir = v
	}
	if v := f.Get("skills-context-file"); v != "" {
		rt.SkillsContextFile = v
	}
	if v := f.Get("usage-format"); v != "" {
		rt.UsageFormat = v // F1 opt-in: "stream-json" captures token actuals
	}
	if v := f.Get("behavioral-preflight"); v != "" {
		if v != store.BehavioralPreflightCodexExecJSONV1 && v != store.BehavioralPreflightCodexExecJSONV2 && v != store.BehavioralPreflightClaudePrintV1 {
			return clikit.Usagef("unsupported --behavioral-preflight %q (supported: %s, %s, %s)", v, store.BehavioralPreflightCodexExecJSONV1, store.BehavioralPreflightCodexExecJSONV2, store.BehavioralPreflightClaudePrintV1)
		}
		rt.BehavioralPreflight = v
		rt.BehavioralPreflightProvenance = store.ProvenanceDeclared
	}
	// Say it at the moment the capability is granted, while the operator is
	// present and can decide. Every outward write dacli itself makes is gated
	// (rw grant, disclosure consent); all of that is bypassed by a child that
	// shells to `gh` directly, which is how an agent once created a repo, set
	// origin, pushed, and merged PRs with nobody approving any of it (308).
	if entry, why := store.UngatedOutwardGrant(append(append([]string{}, rt.Args...), rt.SandboxRO...)); entry != "" {
		fmt.Fprintf(ctx.Stderr, "warning: this runtime's allowlist includes %s — %s. A child on it can write outward without passing dacli's consent gates.\n", entry, why)
	}
	if err := store.CreateRuntime(w, id.ID, rt, ""); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "runtime %s added (binary: %s, mode: %s) — run `dacli runtime doctor` to probe it\n", rt.Name, rt.Binary, rt.Mode)
	return nil
}

// deniedEnvPassthrough returns the first env name a runtime must never forward
// to a child, or "" if all are allowed. The default preset already omits
// ANTHROPIC_API_KEY (children use the operator's own Claude Code login); this
// makes that a checked invariant rather than a value one edit away from being
// undone, closing the "rebuild the sandbox via runtime add" escalation.
func deniedEnvPassthrough(names []string) string {
	denied := map[string]bool{
		"ANTHROPIC_API_KEY":          true,
		"ANTHROPIC_AUTH_TOKEN":       true,
		"ANTHROPIC_BEDROCK_BASE_URL": true,
		"ANTHROPIC_BASE_URL":         true,
		"AWS_SECRET_ACCESS_KEY":      true,
		"AWS_ACCESS_KEY_ID":          true,
		"AWS_SESSION_TOKEN":          true,
		"GITHUB_TOKEN":               true,
		"GH_TOKEN":                   true,
		"OPENAI_API_KEY":             true,
	}
	for _, n := range names {
		u := strings.ToUpper(strings.TrimSpace(n))
		if denied[u] {
			return n
		}
		// Suffix matching catches the credentials nobody thought to enumerate.
		// A fixed list only ever denies yesterday's secret names: the previous
		// one missed every AWS key but the secret, both GitHub tokens, and
		// named ANTHROPIC_BEDROCK_BASE, which is not a real variable (the real
		// one is ..._BASE_URL), so it denied nothing at all.
		for _, suffix := range []string{"_API_KEY", "_TOKEN", "_SECRET", "_SECRET_KEY", "_PASSWORD", "_CREDENTIALS"} {
			if strings.HasSuffix(u, suffix) {
				return n
			}
		}
	}
	return ""
}

func cmdRuntimeList(ctx *clikit.Ctx, args []string) error {
	// This command takes no flags, so ANY flag is a typo. An empty allowlist
	// rejects every one — without it a mistyped flag was dropped and the
	// command ran as if nothing were wrong.
	if f, ferr := clikit.ParseFlags(args); ferr != nil {
		return ferr
	} else if err := f.Reject(); err != nil {
		return err
	}
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	rts, _ := store.LoadRuntimes(w)
	for _, rt := range rts {
		sandbox := "no read-only mode"
		if len(rt.SandboxRO) > 0 {
			sandbox = "ro: " + strings.Join(rt.SandboxRO, " ")
		}
		ceilingSupport := "tokens: no ceiling"
		if rt.TokenLimitFlag != "" {
			ceilingSupport = "tokens: enforced via " + rt.TokenLimitFlag
		}
		fmt.Fprintf(ctx.Stdout, "%-14s %-16s %-6s harness=%s · %s · %s · %s\n", rt.Name, rt.Binary, rt.Mode, rt.Harness, sandbox, ceilingSupport, store.ContextSummary(rt))
	}
	return nil
}

// cmdRuntimeDoctor keeps binary/version and local sandbox evidence separate
// from the optional bounded launch handshake. Vendor-specific declarations
// dacli cannot reason about remain declared/unsupported, never verified.
func cmdRuntimeDoctor(ctx *clikit.Ctx, args []string) error {
	f, ferr := clikit.ParseFlags(args)
	if ferr != nil {
		return ferr
	} else if err := f.Reject("runtime", "grant"); err != nil {
		return err
	}
	grantFilter := model.Grant(f.Get("grant"))
	if grantFilter != "" && grantFilter != model.GrantRO && grantFilter != model.GrantRW {
		return clikit.Usagef("usage: dacli runtime doctor [--runtime name] [--grant ro|rw]")
	}
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	rts, _ := store.LoadRuntimes(w)
	if len(rts) == 0 {
		fmt.Fprintln(ctx.Stdout, "no runtimes configured; `dacli runtime add <name> --preset ...`")
		return nil
	}
	type doctorRecord struct {
		Runtime            string                       `json:"runtime"`
		Binary             string                       `json:"binary"`
		Version            string                       `json:"version,omitempty"`
		Presence           string                       `json:"presence"`
		Grant              model.Grant                  `json:"grant"`
		Launch             store.RuntimeLaunchPreflight `json:"launch"`
		Strategy           string                       `json:"behavioral_preflight,omitempty"`
		StrategyProvenance store.CapabilityProvenance   `json:"behavioral_preflight_provenance,omitempty"`
	}
	var records []doctorRecord
	humanOut := ctx.Stdout
	if ctx.JSON {
		ctx.Stdout = io.Discard
		defer func() { ctx.Stdout = humanOut }()
	}
	for _, rt := range rts {
		if f.Get("runtime") != "" && rt.Name != f.Get("runtime") {
			continue
		}
		grant := grantFilter
		if grant == "" {
			grant = model.GrantRO
			if strings.HasSuffix(rt.Name, "-rw") {
				grant = model.GrantRW
			}
		}
		path, lerr := exec.LookPath(rt.Binary)
		if lerr != nil {
			fmt.Fprintf(ctx.Stdout, "%-14s ✗ binary %q not found on PATH\n", rt.Name, rt.Binary)
			records = append(records, doctorRecord{Runtime: rt.Name, Binary: rt.Binary, Presence: "missing", Grant: grant,
				Launch: store.RuntimeLaunchPreflight{State: store.LaunchUnsupported, Provenance: store.ProvenanceDeclared, Detail: "binary missing"}})
			continue
		}
		version := "version unknown"
		cctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		versionCmd := exec.CommandContext(cctx, path, "--version")
		versionCmd.Dir = w.Root
		if out, verr := commandresult.Run(versionCmd, commandresult.RunOptions{Operation: "runtime version probe", WorkspaceRoot: w.Root, TimedOut: func() bool { return cctx.Err() == context.DeadlineExceeded }}); verr == nil {
			version = strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
			if len(version) > 40 {
				version = version[:40]
			}
		}
		cancel()
		sandbox := "sandbox probe failed (no read-only mode declared; ro spawns will be refused)"
		switch {
		case len(rt.SandboxRO) == 0:
			rt.ROProbe = store.RuntimeROFailed
		case !store.RuntimeROProbeable(rt):
			rt.ROProbe = store.RuntimeROUnknown
			sandbox = "sandbox unknown (declared, not probeable; ro spawns require --cooperative)"
		default:
			if filepath.Base(rt.Binary) == "codex" && hasCodexReadOnly(rt.SandboxRO) {
				probeDir, perr := os.MkdirTemp("", "dacli-codex-ro-*")
				if perr != nil {
					return perr
				}
				defer func() { _ = os.RemoveAll(probeDir) }()
				target := filepath.Join(probeDir, "must-not-write")
				probeCtx, probeCancel := context.WithTimeout(context.Background(), 3*time.Second)
				// Codex exposes its no-model policy runner as `codex sandbox`.
				// The shell prints a marker only after its redirection was attempted.
				// Requiring that marker distinguishes the intended read-only denial
				// from an outer sandbox-exec startup failure, which can emit the same
				// generic "Operation not permitted" text without running our command.
				touchPath, touchErr := exec.LookPath("touch")
				if touchErr != nil {
					probeCancel()
					return fmt.Errorf("locate touch for Codex sandbox probe: %w", touchErr)
				}
				probeScript := `"$1" "$2"; status=$?; printf 'dacli-codex-ro-command-ran:%s\n' "$status"; exit "$status"`
				probeCmd := exec.CommandContext(probeCtx, path,
					"sandbox", "-P", ":read-only", "-C", probeDir, "--",
					"/bin/sh", "-c", probeScript, "dacli-codex-ro-probe", touchPath, target)
				probeCmd.Dir = w.Root
				probeOut, probeErr := commandresult.Run(probeCmd, commandresult.RunOptions{Operation: "Codex read-only sandbox probe", WorkspaceRoot: w.Root, TimedOut: func() bool { return probeCtx.Err() == context.DeadlineExceeded }})
				probeCancel()
				_, statErr := os.Stat(target)
				if probeErr != nil && os.IsNotExist(statErr) && codexSandboxRefusedWrite(probeOut) {
					rt.ROProbe = store.RuntimeROVerified
					sandbox = "sandbox verified (local codex sandbox refused a write)"
				} else {
					rt.ROProbe = store.RuntimeROFailed
					sandbox = "sandbox probe failed (read-only probe did not refuse write): " + strings.TrimSpace(string(probeOut))
				}
				if err := store.SaveRuntimeROProbe(w, rt, path, rt.ROProbe, sandbox); err != nil {
					return err
				}
				break
			}
			probeCtx, probeCancel := context.WithTimeout(context.Background(), 3*time.Second)
			probeArgs := append(append([]string{}, rt.SandboxRO...), "--help")
			probeCmd := exec.CommandContext(probeCtx, path, probeArgs...)
			probeCmd.Dir = w.Root
			probeOut, probeErr := commandresult.Run(probeCmd, commandresult.RunOptions{Operation: "runtime sandbox help probe", WorkspaceRoot: w.Root, TimedOut: func() bool { return probeCtx.Err() == context.DeadlineExceeded }})
			probeCancel()
			missing := runtimeHelpMissing(rt, string(probeOut))
			if probeErr == nil && len(missing) == 0 {
				rt.ROProbe = store.RuntimeROVerified
				sandbox = "sandbox verified (local help accepted and advertised the preset contract)"
			} else {
				rt.ROProbe = store.RuntimeROFailed
				detail := strings.TrimSpace(string(probeOut))
				if probeErr == nil && len(missing) > 0 {
					detail = "help did not advertise " + strings.Join(missing, ", ")
				} else if detail == "" {
					detail = probeErr.Error()
				}
				sandbox = "sandbox probe failed (ro spawns will be refused): " + detail
			}
			if err := store.SaveRuntimeROProbe(w, rt, path, rt.ROProbe, sandbox); err != nil {
				return fmt.Errorf("cache sandbox probe for runtime %s: %w", rt.Name, err)
			}
		}
		fmt.Fprintf(ctx.Stdout, "%-14s ✓ %s · %s · %s\n", rt.Name, path, version, sandbox)
		fmt.Fprintf(ctx.Stdout, "%-14s   %s\n", rt.Name, runtimeContractSummary(rt))
		if contractErr := store.ValidateContextContract(rt); contractErr != nil {
			fmt.Fprintf(ctx.Stdout, "%-14s ✗ %v\n", rt.Name, contractErr)
		} else {
			if probeErr := probeContextDiscovery(rt); probeErr != nil {
				fmt.Fprintf(ctx.Stdout, "%-14s ✗ context discovery probe failed: %v\n", rt.Name, probeErr)
			} else {
				fmt.Fprintf(ctx.Stdout, "%-14s   %s · fixture discovery verified\n", rt.Name, store.ContextSummary(rt))
			}
		}
		// A claude-family binary with no usage_format silently leaves both
		// `agents --tail` and calibration blind (§ 23) — worth flagging even
		// though it's a declared choice, not a probe failure.
		if rt.Binary == "claude" && rt.UsageFormat == "" {
			fmt.Fprintf(ctx.Stdout, "%-14s ⚠ no usage_format: `--tail` and calibration will be blind — enable stream-json\n", rt.Name)
		}
		launch, launchErr := launchCompatibility(ctx, w, rt, path, grant, "", false, true)
		if launchErr != nil {
			return launchErr
		}
		fmt.Fprintf(ctx.Stdout, "%-14s   launch[%s] strategy=%s/%s · result=%s/%s · %s · command %s\n", rt.Name, grant, rt.BehavioralPreflightProvenance, clikit.OrDash(rt.BehavioralPreflight), launch.Provenance, launch.State, launch.Detail, launch.CommandTimestamp.Format(time.RFC3339))
		if rt.BehavioralPreflight == "" {
			fmt.Fprintf(ctx.Stdout, "%-14s   migrate: recreate an exact supported adapter with `dacli runtime add %s-migrated --preset codex-rw` (custom adapters cannot be inferred)\n", rt.Name, rt.Name)
		}
		records = append(records, doctorRecord{Runtime: rt.Name, Binary: path, Version: version, Presence: "present", Grant: grant, Launch: launch, Strategy: rt.BehavioralPreflight, StrategyProvenance: rt.BehavioralPreflightProvenance})
	}
	if f.Get("runtime") != "" && len(records) == 0 {
		return store.ErrNotFound{Ref: "runtime " + f.Get("runtime")}
	}
	if ctx.JSON {
		ctx.Stdout = humanOut
		return clikit.EmitJSON(ctx, records)
	}
	return nil
}

// probeContextDiscovery is intentionally behavioral: an invalid fixture must
// be found under a synthetic home/config root. Merely observing that a vendor
// accepts an ignore flag was the false proof behind issue #691.
func probeContextDiscovery(rt store.Runtime) error {
	home, err := os.MkdirTemp("", "dacli-context-probe-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(home) }()
	work := filepath.Join(home, "repo")
	fixture := filepath.Join(work, "AGENTS.md")
	if err := os.MkdirAll(work, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(fixture, []byte("deliberately invalid context fixture"), 0o600); err != nil {
		return err
	}
	env := map[string]string{"HOME": home, "XDG_CONFIG_HOME": filepath.Join(home, ".config"), "CODEX_HOME": filepath.Join(home, ".codex")}
	for _, source := range store.DiscoverContextSources(rt, work, env) {
		if source.Class == store.ContextRepoInstructions && source.Path == fixture {
			return nil
		}
	}
	return fmt.Errorf("deliberately invalid repository instruction was not enumerated")
}

func runtimeHelpMissing(rt store.Runtime, help string) []string {
	required := []string{"--allowedTools"}
	switch rt.UsageFormat {
	case "gemini-stream-json":
		required = []string{"--prompt", "--model", "--output-format", "--approval-mode", "plan"}
	case "copilot-json":
		required = []string{"--prompt", "--model", "--output-format", "--deny-tool"}
	}
	normalized := strings.ToLower(strings.ReplaceAll(help, "-", ""))
	var missing []string
	for _, flag := range required {
		if !strings.Contains(normalized, strings.ToLower(strings.ReplaceAll(flag, "-", ""))) {
			missing = append(missing, flag)
		}
	}
	return missing
}

func hasCodexReadOnly(args []string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--sandbox" && args[i+1] == "read-only" {
			return true
		}
	}
	return false
}

func codexSandboxRefusedWrite(out []byte) bool {
	detail := strings.ToLower(string(out))
	commandRanAndFailed := false
	for _, line := range strings.Split(detail, "\n") {
		const marker = "dacli-codex-ro-command-ran:"
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, marker) {
			continue
		}
		status, err := strconv.Atoi(strings.TrimPrefix(line, marker))
		commandRanAndFailed = err == nil && status > 0
	}
	if !commandRanAndFailed {
		return false
	}
	for _, marker := range []string{"operation not permitted", "permission denied", "sandbox denial", "access is denied"} {
		if strings.Contains(detail, marker) {
			return true
		}
	}
	return false
}

// cmdSpawn launches a child agent: identity minted, brief assembled and
// FROZEN (the P3 replay capture), sandbox flags applied, process run to
// completion, everything written to a run record. Single-turn by design —
// the small-task assumption is the design center.
// claimTask stamps "claimed by <childID>" onto the task's Log at spawn so a
// claim->completed span exists for that task. calibration.logSpan reads the
// most recent "claimed by" stamp before completion as the span start; without
// it calibrate's by-agent band has no span to join run records against and
// stays empty on real runs (D1). Idempotency is scoped to THIS child: repeated
// turns for one supervised child do not churn the log, while a later spawn must
// append its newly minted identity instead of inheriting a retired same-role
// child's attribution (issue #725). The task is loaded from the shared root,
// so the stamp lands there and travels with the task.
