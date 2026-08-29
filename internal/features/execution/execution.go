// Package execution is the runtime slice: dacli launching agents. Adapter
// management, single spawns, the supervision loop, and run records. This is
// the one slice that runs processes — and where the permission model stops
// being cooperative for spawned children (RUNTIMES.md § 8).
package execution

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/agentstate"
	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/providerpolicy"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "runtime add", Brief: "Add a coding-agent CLI adapter (--preset claude-code|claude-code-rw|codex|codex-rw|gemini|gemini-rw|copilot|copilot-rw|generic-exec)", Mutates: true, Usage: "dacli runtime add <name> [--preset claude-code|claude-code-rw|codex|codex-rw|gemini|gemini-rw|copilot|copilot-rw|generic-exec] [--harness family] [--binary b] [--mode stdin|arg] [--flag -p] [--arg a]... [--sandbox-ro-arg a]... [--env NAME]... [--model-flag f] [--token-limit-flag f] [--execution-capability local-coordination-socket]... [--behavioral-preflight codex-exec-json-v1|codex-exec-json-v2|claude-print-v1]", Run: cmdRuntimeAdd},
	{Path: "runtime rm", Brief: "Remove a runtime adapter (refuses while a role routes to it)", Mutates: true, Usage: "dacli runtime rm <name>", Run: cmdRuntimeRm},
	{Path: "runtime list", Brief: "Configured runtimes and their declared capabilities", Usage: "dacli runtime list", Run: cmdRuntimeList},
	{Path: "runtime doctor", Brief: "Probe binary/version and exact behavioral launch compatibility", JSON: true, Usage: "dacli runtime doctor [--runtime name] [--grant ro|rw]", Run: cmdRuntimeDoctor},
	{Path: "spawn", Brief: "Launch a child agent on a runtime: identity, brief, sandbox, run record (--detach to background)", Mutates: true, Usage: "dacli spawn --task <ref> [--runtime name] [--role r] [--grant ro|rw] [--model m] [--harness family]... [--worktree] [--detach] [--claim path,path] [--pr] [--review [--structured-review-result] [--pr-number N]] [--budget N] [--max-tokens N [--allow-advisory-tokens]] [--timeout sec] [--cooperative|--allow-user-config] [--advise] [--force]", Run: cmdSpawn},
	{Path: "wait", Brief: "Block until detached run(s) finish, then finalize their outcome (default: all live)", Mutates: true, Usage: "dacli wait [<run-id>...] [--interval DUR] [--timeout DUR]", Run: cmdWait},
	{Path: "supervise", Brief: "Spawn-evaluate-correct loop until accepted or --max-turns", Mutates: true, Usage: "dacli supervise --task <ref> [--runtime name] [--role r] [--max-turns N] [--grant ro|rw] [--model m] [--claim path,path] [--pr] [--review [--pr-number N]] [--budget N] [--max-tokens N [--allow-advisory-tokens]] [--timeout sec] [--cooperative|--allow-user-config] [--advise] [--force]", Run: cmdSupervise},
	{Path: "runs list", Brief: "Recorded agent runs, newest first", Usage: "dacli runs list", Run: cmdRunsList},
	{Path: "runs show", Brief: "Invocation, outcome, brief, and transcript for one run", Usage: "dacli runs show <run-id-prefix>", Run: cmdRunsShow},
	{Path: "runs prune", Brief: "Bound transcript growth (--keep N, default 20)", Mutates: true, Usage: "dacli runs prune [--keep N]", Run: cmdRunsPrune},
	{Path: "agents", Brief: "Live spawned agents + RAM/CPU/GPU/state; --project/--json presents sourced worker progress", JSON: true, Usage: "dacli agents [--project slug] [--max-rss MB] [--max-runtime DUR] [--reap] [--tail]", Run: cmdAgents},
	{Path: "logs", Brief: "Print or follow (-f) a run's transcript as it streams", Usage: "dacli logs <run-id-prefix|child-id> [-f] [--tail N]", Run: cmdLogs},
	{Path: "kill", Brief: "Terminate an agent and its ENTIRE process tree (SIGTERM→SIGKILL); reaps runaways", Mutates: true, Usage: "dacli kill <run-id-prefix | child-id> [--grace sec]  |  dacli kill --all", Run: cmdKill},
}

func contextContract(user, repo, skills, plugins, mcp, env store.ContextCapability) map[store.ContextClass]store.ContextCapability {
	return map[store.ContextClass]store.ContextCapability{
		store.ContextUserConfig: user, store.ContextRepoInstructions: repo,
		store.ContextGlobalSkills: skills, store.ContextPlugins: plugins,
		store.ContextMCP: mcp, store.ContextEnvironment: env,
	}
}

// presets are shipped adapters. Their flags are ASSUMPTIONS, recorded as
// such in the adapter body, to be corrected by `runtime doctor` on a machine
// where the binary exists.
var presets = map[string]store.Runtime{
	"claude-code": {
		Name: "claude-code", Harness: "claude", Binary: "claude", Mode: "arg", Flag: "-p",
		// Read-only means read tools plus Bash scoped to the dacli binary —
		// plan mode would mute the child entirely (no Bash = no reporting).
		SandboxRO: []string{"--allowedTools", "Read,Grep,Glob,LS,Bash(dacli:*)"},
		// Deliberately NO ANTHROPIC_API_KEY: children run as the user's own
		// Claude Code login (keychain via HOME/USER), never API billing. If
		// that variable leaked through, billing would silently flip.
		Env:             []string{"HOME", "PATH", "USER", "LOGNAME", "TMPDIR"},
		ModelFlag:       "--model", // role-level cost routing: reviewer=opus, junior=haiku
		SkillsNativeDir: ".claude/skills",
		// Defaults ON: without it, `agents --tail` reads a transcript that
		// stays empty until the child exits (claude buffers stdout under
		// --print), and calibration gets no usage actuals at all (§ 23).
		UsageFormat:         "stream-json",
		BehavioralPreflight: store.BehavioralPreflightClaudePrintV1,
		Context:             contextContract(store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextIsolated),
	},
	// claude-code-rw is the write-capable counterpart. The stock claude-code
	// preset only declares SandboxRO, so an rw spawn on it was refused with
	// "grants no write tool" and no obvious remedy — the operator had to
	// hand-build a second runtime with the right --allowedTools list to do the
	// most ordinary thing dacli does, which is run an implementer (task 309,
	// issue #382 item 7).
	//
	// The allowlist is the read set plus Edit/Write and the two commands an
	// implementer cannot work without: git for its own branch and commits, and
	// the dacli binary for claiming, checking and reporting. Everything else
	// stays out; widen it per-runtime, deliberately, rather than here.
	"claude-code-rw": {
		Name: "claude-code-rw", Harness: "claude", Binary: "claude", Mode: "arg", Flag: "-p",
		Args: []string{"--allowedTools", "Read,Grep,Glob,LS,Edit,Write,Bash(git:*),Bash(dacli:*)"},
		// Same read-only list as claude-code, so one runtime can serve both
		// grants: a ro spawn on this adapter is still sandboxed to reads.
		SandboxRO:           []string{"--allowedTools", "Read,Grep,Glob,LS,Bash(dacli:*)"},
		Env:                 []string{"HOME", "PATH", "USER", "LOGNAME", "TMPDIR"},
		ModelFlag:           "--model",
		SkillsNativeDir:     ".claude/skills",
		UsageFormat:         "stream-json",
		BehavioralPreflight: store.BehavioralPreflightClaudePrintV1,
		Context:             contextContract(store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextIsolated),
	},
	"generic-exec": {
		Name: "generic-exec", Harness: "generic", Binary: "", Mode: "stdin",
		Env:     []string{"HOME", "PATH"},
		Context: contextContract(store.ContextUnsupported, store.ContextEnumerated, store.ContextUnsupported, store.ContextUnsupported, store.ContextUnsupported, store.ContextUnsupported),
	},
	"codex": {
		Name: "codex", Harness: "codex", Binary: "codex", Mode: "stdin",
		GlobalArgs: []string{"--ask-for-approval", "never"},
		Args:       []string{"exec", "--json", "--ephemeral", "--color", "never", "--sandbox", "read-only"},
		SandboxRO:  []string{"--sandbox", "read-only"},
		Env:        []string{"HOME", "PATH", "USER", "LOGNAME", "TMPDIR", "CODEX_HOME"},
		ModelFlag:  "--model", UsageFormat: "codex-jsonl",
		BehavioralPreflight: store.BehavioralPreflightCodexExecJSONV2,
		Context:             contextContract(store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextIsolated),
	},
	"codex-rw": {
		Name: "codex-rw", Harness: "codex", Binary: "codex", Mode: "stdin",
		GlobalArgs: []string{"--ask-for-approval", "never"},
		Args:       []string{"exec", "--json", "--ephemeral", "--color", "never", "--sandbox", "workspace-write"},
		SandboxRO:  []string{"--sandbox", "read-only"},
		Env:        []string{"HOME", "PATH", "USER", "LOGNAME", "TMPDIR", "CODEX_HOME"},
		ModelFlag:  "--model", UsageFormat: "codex-jsonl",
		BehavioralPreflight: store.BehavioralPreflightCodexExecJSONV2,
		Context:             contextContract(store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextIsolated),
	},
	"gemini": {
		Name: "gemini", Harness: "gemini", Binary: "gemini", Mode: "arg", Flag: "-p",
		SandboxRO:       []string{"--approval-mode", "plan"},
		Env:             []string{"HOME", "PATH", "USER", "LOGNAME", "TMPDIR"},
		ModelFlag:       "--model",
		SkillsNativeDir: ".gemini/skills",
		UsageFormat:     "gemini-stream-json",
		Context:         contextContract(store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextIsolated),
	},
	"gemini-rw": {
		Name: "gemini-rw", Harness: "gemini", Binary: "gemini", Mode: "arg", Flag: "-p",
		// auto_edit permits workspace edits but does not silently approve every
		// shell command or MCP call. That boundary is why this is not --yolo.
		Args:            []string{"--approval-mode", "auto_edit"},
		SandboxRO:       []string{"--approval-mode", "plan"},
		Env:             []string{"HOME", "PATH", "USER", "LOGNAME", "TMPDIR"},
		ModelFlag:       "--model",
		SkillsNativeDir: ".gemini/skills",
		UsageFormat:     "gemini-stream-json",
		Context:         contextContract(store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextEnumerated, store.ContextIsolated),
	},
	"copilot": {
		Name: "copilot", Harness: "copilot", Binary: "copilot", Mode: "arg", Flag: "-p",
		SandboxRO:   []string{"--deny-tool", "write", "--deny-tool", "shell"},
		Env:         []string{"HOME", "PATH", "USER", "LOGNAME", "TMPDIR"},
		ModelFlag:   "--model",
		UsageFormat: "copilot-json",
		Context:     contextContract(store.ContextEnumerated, store.ContextEnumerated, store.ContextUnsupported, store.ContextEnumerated, store.ContextEnumerated, store.ContextIsolated),
	},
	"copilot-rw": {
		Name: "copilot-rw", Harness: "copilot", Binary: "copilot", Mode: "arg", Flag: "-p",
		// There is deliberately no --allow-all-tools. The child may edit and
		// run only the two command families needed by the dacli work loop.
		Args:        []string{"--allow-tool", "write", "--allow-tool", "shell(git:*)", "--allow-tool", "shell(dacli:*)"},
		SandboxRO:   []string{"--deny-tool", "write", "--deny-tool", "shell"},
		Env:         []string{"HOME", "PATH", "USER", "LOGNAME", "TMPDIR"},
		ModelFlag:   "--model",
		UsageFormat: "copilot-json",
		Context:     contextContract(store.ContextEnumerated, store.ContextEnumerated, store.ContextUnsupported, store.ContextEnumerated, store.ContextEnumerated, store.ContextIsolated),
	},
}

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
func claimTask(ctx *clikit.Ctx, w *workspace.Workspace, t *store.Task, childID string) {
	err := store.WithTask(w, t, func(fresh *store.Task) error {
		if store.ClaimedBy(fresh) == childID {
			return nil
		}
		store.AppendLog(fresh, "claimed by "+childID)
		return store.SaveTask(fresh)
	})
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "warning: could not stamp claim on task %03d-%s: %v\n", t.Seq, t.Slug, err)
	}
}

func contextInvocation(role team.Role, hasRole, override bool, sources []store.ContextSource) string {
	skills := "-"
	if hasRole && len(role.Skills) > 0 {
		skills = strings.Join(role.Skills, ",")
	}
	lines := []string{"declared_role_skills: " + skills, fmt.Sprintf("external_context_override: %t", override)}
	if len(sources) == 0 {
		lines = append(lines, "external_context_sources: -")
	}
	for _, source := range sources {
		lines = append(lines, fmt.Sprintf("external_context_source: %s=%s", source.Class, source.Path))
	}
	return strings.Join(lines, "\n") + "\n"
}

// --- The shared pre-launch path ---
//
// Every command that puts a brief in front of a runtime and starts a child —
// `spawn` and `supervise` — takes the SAME prologue: resolve role/runtime/
// grant/band, run every governance gate, build the sandbox. It is deliberately
// ONE function over ONE gate list, not a convention two commands are trusted to
// follow: `supervise` used to reimplement the prologue by hand and silently
// dropped four gates (role WIP, taint blast-radius, --max-tokens, --claim
// overlap), so a brief that `spawn` refused could be pushed through the same
// runtime for up to --max-turns turns just by typing the other verb.
//
// Adding a gate therefore means adding an entry to launchGates, which is the
// only place a gate CAN be added — there is no per-command prologue left to add
// it to, and no per-command gate to forget.
//
// What genuinely stays per-command is the part AFTER the gates: `spawn` owns
// --detach and --worktree (a single run, optionally backgrounded or isolated in
// its own git worktree); `supervise` owns --max-turns and the evaluate-correct
// loop. Those are execution shapes, not policy decisions, so they are not gates
// and are not shared.

// launchPlan is everything decided before an identity is minted or a process
// starts. Both commands build it via resolveLaunch and read their settings from
// it rather than re-deriving any of them.
type launchPlan struct {
	Task                 *store.Task
	TaskRef              string
	RoleName             string
	Role                 team.Role
	HasRole              bool
	Grant                model.Grant
	Model                string
	Runtime              store.Runtime
	Band                 store.Band
	Claims               []string
	Sandbox              []string
	Budget               int
	Timeout              int
	ContextSources       []store.ContextSource
	ContextOverride      bool
	MutationCapabilities []mutationCapabilityResult
	PlannedHandoffs      []string
	ProbeWorkDir         string

	w *workspace.Workspace
	f *clikit.Flags
}

// launchGate is one pre-launch policy decision: it reads the resolved plan and
// either passes or REFUSES (exit 3). A gate never mutates the plan and never
// starts anything, so every refusal below lands before any process exists.
type launchGate struct {
	Name  string
	Check func(ctx *clikit.Ctx, p *launchPlan) error
}

// launchGates is THE gate list. resolveLaunch runs it in order for every
// spawning command; nothing else enforces a spawn-time policy.
var launchGates = []launchGate{
	{"role-wip", gateRoleWIP},
	{"seniority", gateSeniority},
	{"phase", gatePhase},
	{"token-budget", gateTokenBudget},
	{"taint", gateTaint},
	{"claim-overlap", gateClaimOverlap},
}

// launchFlags are the flags the shared prologue itself reads. EVERY spawning
// command must accept them: a command whose flag set omits --claim or
// --max-tokens cannot be gated by them at all, and one that omits --force
// cannot be un-gated deliberately, so a missing flag is a missing gate by
// another route. Commands add their own flags on top via launchFlagsWith.
var launchFlags = []string{
	"task", "runtime", "role", "grant", "model", "harness",
	"claim", "budget", "max-tokens", "timeout",
	"cooperative", "allow-user-config", "advise", "force", "allow-advisory-tokens",
}

// launchFlagsWith returns the shared flag set plus a command's own flags,
// copying so no caller can append into the package-level slice.
func launchFlagsWith(extra ...string) []string {
	return append(append([]string{}, launchFlags...), extra...)
}

// errAdviseOnly is the sentinel resolveLaunch returns when `--advise` fired:
// the advisory was printed and the launch must NOT proceed. Callers unwrap it to
// a clean exit-0 preview (dacli 232) — `--advise` previews on `spawn`/`supervise`
// exactly as it does on `loop`, so a flag named "advise" never launches a run.
var errAdviseOnly = errors.New("advise-only: previewed, nothing launched")

// resolveLaunch turns flags into a fully-gated launchPlan: role defaults,
// runtime resolution, the advisory readout, every gate in launchGates, and the
// sandbox. It returns only plans that are cleared to launch. A `--advise` preview
// short-circuits with errAdviseOnly before any gate runs — nothing is minted.
func resolveLaunch(ctx *clikit.Ctx, w *workspace.Workspace, f *clikit.Flags, taskRef string) (*launchPlan, error) {
	t, err := store.FindTask(w, taskRef)
	if err != nil {
		return nil, err
	}
	// Role defaults: grant, runtime routing, model tier. The role's WIP and
	// seniority/phase caps are enforced below as gates, not here.
	p := &launchPlan{
		Task: t, TaskRef: taskRef, w: w, f: f,
		RoleName: f.Get("role"),
		Grant:    model.Grant(f.Get("grant")),
		Model:    f.Get("model"),
	}
	rtName := f.Get("runtime")
	if role, ok := store.LoadRole(w, p.RoleName); ok {
		p.Role, p.HasRole = role, true
		if p.Grant == "" && role.Grant != "" {
			p.Grant = model.Grant(role.Grant)
		}
		if rtName == "" {
			rtName = role.Runtime
		}
		if p.Model == "" {
			p.Model = role.Model
		}
	}
	if p.Grant == "" {
		p.Grant = model.GrantRO
	}
	if rtName == "" {
		return nil, clikit.Usagef("no runtime: pass --runtime or set `runtime:` on the role")
	}
	// An explicit --runtime is an operator choice and is never substituted.
	// Role routing may use only that role's named, ordered fallback chain.
	limits := store.LoadRuntimeLimits(w)
	if cooldown, open, limitErr := limits.Open(rtName); limitErr != nil {
		return nil, fmt.Errorf("runtime %s cooldown: %w", rtName, limitErr)
	} else if open {
		remaining := time.Until(cooldown.Until)
		if remaining < 0 {
			remaining = 0
		}
		destination := ""
		if f.Get("runtime") == "" && p.HasRole && (providerpolicy.Outcome{Kind: cooldown.Kind}).Fallbackable() {
			roles, loadErr := store.LoadRoles(w)
			if loadErr != nil {
				return nil, loadErr
			}
			source := p.Role
			source.Grant = string(p.Grant)
			allowedHarnesses := f.All("harness")
			allowedFallback := func(role team.Role) bool {
				if len(allowedHarnesses) == 0 {
					return true
				}
				rt, loadErr := store.LoadRuntime(w, role.Runtime)
				return loadErr == nil && slices.Contains(allowedHarnesses, rt.Harness)
			}
			if fallback, _, ok, selectErr := store.SelectFallbackMatching(source, roles, limits, allowedFallback); selectErr != nil {
				return nil, selectErr
			} else if ok {
				destination = fallback.Runtime
				p.Role, p.HasRole, p.RoleName = fallback, true, fallback.Name
				rtName = fallback.Runtime
				if f.Get("grant") == "" && fallback.Grant != "" {
					p.Grant = model.Grant(fallback.Grant)
				}
				if f.Get("model") == "" {
					p.Model = fallback.Model
				}
			}
		}
		transition := providerpolicy.Transition{Source: cooldown.Runtime, Destination: destination, Reason: cooldown.Reason, Cooldown: remaining}
		if reportErr := limits.Report(ctx.Stdout, transition); reportErr != nil {
			return nil, reportErr
		}
		if destination == "" {
			return nil, clikit.Refusedf("runtime %s is paused for %s (%s)", cooldown.Runtime, remaining, cooldown.Reason)
		}
	}
	rt, err := store.LoadRuntime(w, rtName)
	if err != nil {
		return nil, err
	}
	path, err := exec.LookPath(rt.Binary)
	if err != nil {
		return nil, fmt.Errorf("runtime %s: binary %q not on PATH — `dacli runtime doctor`", rt.Name, rt.Binary)
	}
	rt = store.HydrateRuntimeROProbe(w, rt, path)
	p.Runtime = rt
	if allowed := f.All("harness"); len(allowed) > 0 && !slices.Contains(allowed, rt.Harness) {
		return nil, clikit.Refusedf("runtime %s belongs to harness %s, outside the allowed harness policy %s; choose a compatible role/runtime or explicitly authorize a hybrid harness set", rt.Name, rt.Harness, strings.Join(allowed, ","))
	}
	// The band is built in the SAME recorded form invocation.txt uses (OrDash
	// for an empty role/model, rt.Name for runtime) so it matches the bands
	// store.CalibrationSamples joins back from the run records.
	p.Band = store.Band{Role: clikit.OrDash(p.RoleName), Model: clikit.OrDash(p.Model), Runtime: rt.Name}
	// Issue #794: Get returns only the final occurrence. Claims are repeatable
	// scope declarations, so losing an earlier value makes the run record lie
	// and causes dacli commit to refuse work the operator explicitly claimed.
	p.Claims = splitClaimValues(f.All("claim"))
	if p.Budget, err = f.IntAliased(0, "brief-tokens", "budget"); err != nil {
		return nil, err
	}
	p.Timeout = 300
	if n, err := f.Int("timeout", 0); err != nil {
		return nil, err
	} else if n > 0 {
		p.Timeout = n
	}

	// --advise: with role/model/runtime/task resolved but BEFORE any identity is
	// minted or process launched, surface what the log already knows at the launch
	// decision — a calibrated sizing for this agent band and this task's taint
	// status — then STOP. `--advise` previews and never acts, identically on
	// `spawn`, `supervise` and `loop` (dacli 232): a flag whose name promises a
	// preview must not silently launch a real run and bill the operator for a
	// spawn they only meant to price. The gates below and the launch never run;
	// the caller unwraps errAdviseOnly to a clean exit-0 preview.
	if f.Bool("advise") {
		printAdvisory(ctx, w, t, p.Band, p.Claims)
		if err := printTokenCeiling(ctx, p, true); err != nil {
			return nil, err
		}
		return nil, errAdviseOnly
	}

	for _, g := range launchGates {
		if err := g.Check(ctx, p); err != nil {
			return nil, err
		}
	}

	// dacli 272: report every preflight mismatch (binary-allowlist,
	// prompt-tools; grant-write is reported here too but sandboxFor below
	// still owns the actual refusal, unchanged) BEFORE sandboxFor can refuse
	// and return early — otherwise a grant-write refusal would hide a
	// binary-allowlist or prompt-tools warning nobody would ever see.
	exe, _ := os.Executable() // "" on error; preflightIssues skips class 2 then, same as the old warnExeAllowlist
	override := f.Bool("cooperative") || f.Bool("allow-user-config")
	for _, iss := range preflightIssues(rt, p.Role, p.HasRole, p.Grant, override, exe) {
		if !iss.refuse {
			fmt.Fprintf(ctx.Stderr, "warning: %s\n", iss.message)
		}
	}
	contextMismatches, sources := contextIssues(rt, p.Role, p.HasRole, override, ctx.Cwd, currentEnvNames())
	var refused []string
	for _, iss := range contextMismatches {
		if iss.refuse {
			refused = append(refused, iss.message)
		} else {
			fmt.Fprintf(ctx.Stderr, "warning: allowing external context: %s\n", iss.message)
		}
	}
	if len(refused) > 0 {
		return nil, clikit.Refusedf("%s; pass --allow-user-config (or --cooperative) to record and allow the exception", strings.Join(refused, "; "))
	}
	p.ContextSources, p.ContextOverride = sources, override

	sandbox, err := sandboxFor(ctx, rt, p.Grant, override)
	if err != nil {
		return nil, err
	}
	p.Sandbox = sandbox
	// Issue #746: this remains inside resolveLaunch, before identity minting,
	// task claims, run records, and worktrees. A local sandbox flag probe alone
	// did not prove Codex could initialize the exact app-server transport.
	if err := requireLaunchCompatibility(ctx, w, rt, path, p.Grant, p.Model, override); err != nil {
		return nil, err
	}
	// Resolve an already-existing assignment checkout for concrete mutation
	// probes. A requested new --worktree does not exist yet; its creation is
	// still proven later by git, so preflight conservatively probes the main
	// checkout rather than inventing success for a path that cannot be opened.
	p.ProbeWorkDir = w.Root
	if candidate, _, resolveErr := resolveSpawnWorkDir(w, t, ctx.Cwd, f.Bool("worktree")); resolveErr != nil {
		return nil, resolveErr
	} else if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
		p.ProbeWorkDir = candidate
	}
	// Issue #874: prove the resolved assignment can perform its required
	// mutations before an identity, worktree, or paid worker exists. Publication
	// capabilities may be intentionally delegated to root; source writes and
	// declared verification tools may not.
	mutationResults, err := mutationPreflight(p)
	if err != nil {
		return nil, err
	}
	p.MutationCapabilities = mutationResults
	p.PlannedHandoffs = plannedHandoffCapabilities(mutationResults)
	for _, result := range mutationResults {
		if result.Disposition == "planned_handoff" {
			fmt.Fprintf(ctx.Stderr, "planned root handoff: %s (%s): %s\n", result.Capability, result.Class, result.Detail)
		}
	}
	return p, nil
}

// exeAllowlistWarning is dacli 267's binary-allowlist check (class 2 of dacli
// 272's preflight, via preflightIssues): it builds the args the child
// actually runs with (invoke args plus the ro sandbox that was applied) and
// reports the warning text, if any. sandbox is empty for an rw grant, so an
// rw child on cc-rw is checked against its invoke_args allowlist; an ro child
// is checked against that plus its sandbox_ro_args.
func exeAllowlistWarning(rt store.Runtime, sandbox []string, exe string) (string, bool) {
	args := append(append([]string{}, rt.Args...), sandbox...)
	ok, allowlisted := store.RuntimeAllowsDacli(args, exe)
	if ok {
		return "", false
	}
	return fmt.Sprintf("warning: runtime %s allowlists the dacli binary at %s, but this dacli is %s — a headless child cannot run the dacli path its own preamble names (dacli 267); re-run `go install ./cmd/dacli` so ~/go/bin/dacli matches, or update the runtime's allowlist\n",
		rt.Name, strings.Join(allowlisted, ", "), exe), true
}

// gateRoleWIP enforces the role's work-in-progress cap. WIP counts LIVE agents
// holding the role, so it binds anything that mints one — a supervised run
// occupies a slot for its whole loop exactly as a spawn does.
func gateRoleWIP(_ *clikit.Ctx, p *launchPlan) error {
	if !p.HasRole || p.Role.WIP <= 0 {
		return nil
	}
	active, err := store.ActiveInRole(p.w, p.RoleName)
	if err != nil {
		// Cannot rule out the role already being at its cap, so this must not
		// read as "nobody holds the role" and wave the spawn through (dacli
		// 341): fail closed, the same rule gateClaimOverlap already holds
		// itself to for the runs dir (dacli 337, "a gate must never certify
		// what it could not read").
		return fmt.Errorf("cannot check role %s WIP: %w", p.RoleName, err)
	}
	if active >= p.Role.WIP {
		// Name the way out. A bare "at its WIP limit" left an operator staring
		// at `dacli agents` reporting nobody live, with no stated path from
		// refusal to running — its sibling refusals all name theirs (task 295).
		return clikit.Refusedf("role %s is at its WIP limit (%d/%d) — `dacli agents` shows the live runs consuming capacity; stop or finish one, or raise the cap: `dacli role bump %s --wip %d`",
			p.RoleName, active, p.Role.WIP, p.RoleName, p.Role.WIP+1)
	}
	return nil
}

func gateSeniority(_ *clikit.Ctx, p *launchPlan) error {
	if !p.HasRole {
		return nil
	}
	return seniorityGate(p.Role, p.Task)
}

func gatePhase(_ *clikit.Ctx, p *launchPlan) error {
	if !p.HasRole {
		return nil
	}
	return phaseGate(p.w, p.Task, p.Role)
}

// gateTokenBudget makes --max-tokens a runtime-enforced launch contract.
// Calibration predicts whether the task is likely to fit; only an adapter's
// declared TokenLimitFlag can impose the actual ceiling. Unsupported runtimes
// fail closed unless the operator explicitly chooses advisory-only accounting.
//
// For `supervise` the figure is the cost of ONE turn, because the band's
// measured cost is per-run and the loop re-sends the brief once per turn: a
// plan that already busts the budget on a single turn cannot possibly fit the
// loop. Scaling it by --max-turns would refuse on a cost that is never incurred
// when the task is accepted on turn 1, so the single-turn figure is the honest
// bound to gate on.
func gateTokenBudget(ctx *clikit.Ctx, p *launchPlan) error {
	return printTokenCeiling(ctx, p, false)
}

func printTokenCeiling(ctx *clikit.Ctx, p *launchPlan, preview bool) error {
	maxTok, set, err := requestedTokenLimit(p)
	if err != nil || !set {
		return err
	}
	enforced := p.Runtime.TokenLimitFlag != ""
	if !enforced && !p.f.Bool("allow-advisory-tokens") && !preview {
		return clikit.Refusedf("runtime %s cannot enforce --max-tokens %d: its adapter declares no token-limit flag — choose a capable runtime or pass --allow-advisory-tokens to launch with accounting that is explicitly advisory only", p.Runtime.Name, maxTok)
	}
	if enforced {
		fmt.Fprintf(ctx.Stderr, "token ceiling: ENFORCED by runtime %s via %s %d\n", p.Runtime.Name, p.Runtime.TokenLimitFlag, maxTok)
	} else if preview && !p.f.Bool("allow-advisory-tokens") {
		fmt.Fprintf(ctx.Stderr, "token ceiling: UNSUPPORTED by runtime %s — launch with --max-tokens %d would refuse; --allow-advisory-tokens explicitly selects advisory-only accounting\n", p.Runtime.Name, maxTok)
	} else {
		fmt.Fprintf(ctx.Stderr, "warning: token ceiling: ADVISORY ONLY — runtime %s cannot enforce %d tokens; launch allowed by explicit --allow-advisory-tokens\n", p.Runtime.Name, maxTok)
	}
	expected, n, ok := bandTokenBudget(p.w, p.Task, p.Band)
	switch {
	case !ok:
		fmt.Fprintf(ctx.Stderr, "calibrated estimate: unavailable for band %s (no measured token cost yet or task unestimated)\n", p.Band)
	case expected <= float64(maxTok):
		fmt.Fprintf(ctx.Stderr, "calibrated estimate: ~%.0f tokens within ceiling %d (n=%d)\n", expected, maxTok, n)
	case n < 10:
		fmt.Fprintf(ctx.Stderr, "calibrated estimate: PROVISIONAL ~%.0f tokens exceeds ceiling %d (n=%d < 10); the task may not fit\n", expected, maxTok, n)
	default:
		fmt.Fprintf(ctx.Stderr, "warning: calibrated estimate ~%.0f tokens exceeds ceiling %d (n=%d); the task may not fit\n", expected, maxTok, n)
	}
	return nil
}

func requestedTokenLimit(p *launchPlan) (int, bool, error) {
	maxStr := p.f.Get("max-tokens")
	if maxStr == "" {
		return 0, false, nil
	}
	maxTok, err := strconv.Atoi(maxStr)
	if err != nil || maxTok <= 0 {
		return 0, false, clikit.Usagef("--max-tokens takes a positive integer")
	}
	return maxTok, true, nil
}

func tokenLimitArgs(p *launchPlan) []string {
	maxTok, set, err := requestedTokenLimit(p)
	if err != nil || !set || p.Runtime.TokenLimitFlag == "" {
		return nil
	}
	return []string{p.Runtime.TokenLimitFlag, strconv.Itoa(maxTok)}
}

func tokenLimitMode(p *launchPlan) string {
	if p.f.Get("max-tokens") == "" {
		return "unset"
	}
	if p.Runtime.TokenLimitFlag != "" {
		return "runtime-enforced"
	}
	return "advisory-only"
}

// gateTaint is the launch-time taint gate (D3): --advise DISPLAYS taint status;
// here it BLOCKS. If this task's brief sits in an external source's blast
// radius, refuse (exit 3) rather than feed a possibly-injected brief to a fresh
// child — taint stops being an audit query you run after the fact and becomes a
// gate at the point of consumption (RUNTIMES §18, cross-tree injection).
// --force (or --cooperative) is the explicit, loud override: the operator has
// read the origins and accepts the risk. A supervised run consumes that same
// brief once per turn, so it is if anything the more exposed of the two.
func gateTaint(_ *clikit.Ctx, p *launchPlan) error {
	origins, inRadius, _ := externalRadius(p.w, p.Task)
	if inRadius && !(p.f.Bool("force") || p.f.Bool("cooperative")) {
		return clikit.Refusedf("task %03d-%s is in the blast radius of %s — its brief may carry injected content (RUNTIMES §18); audit the origins, then re-run with --force to spawn anyway",
			p.Task.Seq, p.Task.Slug, strings.Join(origins, ", "))
	}
	return nil
}

// gateClaimOverlap enforces the --claim disjointness that keeps parallel
// branches merge-clean: if a live agent already claims an overlapping tree,
// refuse instead of hoping the two never collide.
func gateClaimOverlap(_ *clikit.Ctx, p *launchPlan) error {
	if len(p.Claims) == 0 {
		return nil
	}
	live, err := liveAgents(p.w)
	if err != nil {
		// Cannot rule out a clash, so this must not read as "nobody is
		// working" and wave the spawn through (dacli 337): fail closed, the
		// same rule internal/gates already holds live agents to for its own
		// quantifier gates ("a gate must never certify what it could not
		// read").
		return fmt.Errorf("cannot check for a claim overlap: %w", err)
	}
	for _, other := range live {
		if mine, theirs, clash := procmon.PathsOverlap(p.Claims, other.Claims); clash {
			return clikit.Refusedf("path-claim conflict: live agent %s already claims %q and you claim %q — narrow your scope, or `dacli wait %s` first",
				other.Child, theirs, mine, other.RunID[:min(10, len(other.RunID))])
		}
	}
	return nil
}

func cmdSpawn(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject(launchFlagsWith("worktree", "detach", "pr", "review", "structured-review-result", "pr-number")...); err != nil {
		return err
	}
	if f.Bool("structured-review-result") && !f.Bool("review") {
		return clikit.Usagef("--structured-review-result requires --review")
	}
	taskRef := f.Get("task")
	if taskRef == "" {
		return clikit.Usagef("usage: dacli spawn --task <ref> [--runtime name] [--role r] [--grant ro|rw] [--model m] [--harness family]... [--worktree] [--detach] [--claim path,path] [--pr] [--review [--structured-review-result] [--pr-number N]] [--budget N] [--max-tokens N [--allow-advisory-tokens]] [--timeout sec] [--cooperative|--allow-user-config] [--advise] [--force]")
	}
	plan, err := resolveLaunch(ctx, w, f, taskRef)
	if errors.Is(err, errAdviseOnly) {
		return nil // --advise previewed and stopped; nothing spawned (dacli 232)
	}
	if err != nil {
		return err
	}
	t, rt := plan.Task, plan.Runtime
	if err := validateReviewTarget(w, t, f); err != nil {
		return err
	}
	roleName, modelName, grant := plan.RoleName, plan.Model, plan.Grant
	claims, sandboxArgs := plan.Claims, plan.Sandbox
	budget, timeout := plan.Budget, plan.Timeout

	childID, token, err := agentid.Spawn(w, id, roleName, grant)
	if err != nil {
		return err
	}
	// Stamp the claim now that the child id is minted: this is the span start
	// calibrate joins run actuals against (D1). Idempotent — a re-spawn respects
	// the existing claim.
	claimTask(ctx, w, t, childID)

	b, err := brief.Assemble(w, taskRef, brief.Options{Budget: budget, Role: roleName})
	if err != nil {
		return err
	}
	suffix, err := promptSuffix(w, f, t, childID, grant)
	if err != nil {
		return err
	}
	prompt := b.Render() + suffix

	// The run record: what was this agent told, exactly (PROPOSALS P3).
	runID := ulid.New()
	ctx.Result = commandresult.Spawn{RunID: runID}
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	record := openRunRecord(runDir, ctx.Stderr)
	if len(plan.PlannedHandoffs) > 0 {
		if err := record.critical("planned-handoffs.txt", strings.Join(plan.PlannedHandoffs, "\n")+"\n"); err != nil {
			return err
		}
	}
	if err := record.critical("brief.md", prompt); err != nil {
		return err
	}

	invocation := fmt.Sprintf("run: %s\ntask: %s\nchild: %s\nrole: %s\nmodel: %s\ngrant: %s\nruntime: %s\nbinary: %s\nenv_names: %s\nbudget: %d (recorded, not enforced: runtime reports no usage)\nmax_tokens: %s\nmax_tokens_mode: %s\ntimeout_s: %d\n",
		runID, t.ID, childID, clikit.OrDash(roleName), clikit.OrDash(modelName), grant, rt.Name, rt.Binary,
		strings.Join(append([]string{agentid.EnvVar}, rt.Env...), ","), budget, f.Get("max-tokens"), tokenLimitMode(plan), timeout)
	invocation += contextInvocation(plan.Role, plan.HasRole, plan.ContextOverride, plan.ContextSources)
	invocation += mutationPreflightInvocation(plan.MutationCapabilities)

	// --worktree isolates this child in its own git worktree + branch, so
	// several children spawned in parallel never clobber each other's working
	// tree. The child edits CODE there; workspace.Find redirects its dacli
	// state (identity, tasks, events) to the shared root, so the child sees its
	// own freshly-minted identity and can self-commit, self-check, self-report
	// — no shadow .dacli. Its events therefore land in the shared root, which
	// is exactly where we read them back from below.
	workDir, isolatedWorktree, err := resolveSpawnWorkDir(w, t, ctx.Cwd, f.Bool("worktree"))
	if err != nil {
		return err
	}
	if !isolatedWorktree {
		// Without --worktree the child works in the MAIN checkout, and its
		// protocol tells it to branch — so the main tree's HEAD moves off
		// trunk and stays there. On a fresh repo that also means trunk is
		// never established at all, and `ship` later reports "integrated 0
		// task(s)" as if nothing needed doing, with nowhere to integrate INTO
		// (issue #382 item 3). Say it here, where it is still cheap to fix,
		// rather than leaving the operator to discover a trunkless repo.
		warnMainCheckoutSpawn(ctx, w)
	}
	if f.Bool("worktree") {
		if !gitx.Available() {
			return fmt.Errorf("--worktree needs git on PATH")
		}
		wtPath := w.WorktreePath(t.Project, t.Seq, t.Slug)
		// Pass trunk so a REUSED branch is fast-forwarded before the child sees
		// it. A recurring task keeps its branch across every run, so without
		// this the agent audits a tree as far behind trunk as the task is old
		// and re-reports defects that were fixed long ago (issue #441).
		freshened, err := gitx.AddWorktree(w.Root, wtPath, fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug), store.TrunkBranch(w))
		if err != nil {
			// An existing worktree (a re-spawn) is fine; a real failure is not.
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
		}
		if freshened {
			fmt.Fprintf(ctx.Stderr, "note: fast-forwarded %s to %s before spawning — it was behind trunk\n",
				fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug), store.TrunkBranch(w))
		}
		workDir = wtPath
		record.bestEffort("worktree.txt", wtPath+"\n")
		// The worktree-sandbox fix: the runtime sandbox allowlists the `dacli`
		// binary at the MAIN checkout's absolute path, so a worktree agent can
		// mistake main for its repo and edit code there — clobbering the main
		// tree and sibling agents. A generic "edit relative to cwd" preamble did
		// not override that signal. State the ACTUAL working directory explicitly
		// and forbid editing outside it; this is what keeps the edits in the
		// worktree. Re-freeze brief.md so the run record matches what was sent.
		prompt += worktreePreamble(wtPath)
		if err := record.critical("brief.md", prompt); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stderr, "isolated worktree: %s\n", wtPath)
	} else if isolatedWorktree {
		// A follow-up launched from the task's existing checkout must carry the
		// same isolation signals as the run that originally created it. Merely
		// fixing cmd.Dir would leave the prompt pointing at the main checkout and
		// disable the escape check below, recreating issue #673 through a
		// different signal.
		record.bestEffort("worktree.txt", workDir+"\n")
		prompt += worktreePreamble(workDir)
		if err := record.critical("brief.md", prompt); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stderr, "resuming task worktree: %s\n", workDir)
	}

	// The break-glass BLOCKED channel (task 269): a plain file the child writes
	// with no dacli, read back by `agents`/`wait`. Appended LAST — after any
	// worktree preamble — so its one carve-out from the worktree rule (this path
	// is the shared record store, not the code tree) is stated last. Re-freeze
	// brief.md so the run record matches exactly what the child was sent.
	prompt += blockedChannelPreamble(filepath.Join(runDir, blockedFileName))
	if grant == model.GrantRW {
		// Keep the structured publication-failure channel after the general
		// worktree and BLOCKED rules so its exact second shared-record path is
		// never contradicted by an earlier "outside the worktree" instruction.
		prompt += handoffChannelPreamble(runDir, plan.PlannedHandoffs)
	}
	if err := record.critical("brief.md", prompt); err != nil {
		return err
	}
	provenance, err := promptInvocation(w.PromptsDir(), prompt)
	if err != nil {
		return err
	}
	if err := record.critical("invocation.txt", invocation+provenance); err != nil {
		return err
	}

	extraArgs := append(append([]string{}, sandboxArgs...), modelArgs(ctx, rt, modelName)...)
	extraArgs = append(extraArgs, tokenLimitArgs(plan)...)
	// A child launched with ungated outward reach is recorded, not merely
	// warned about. The warning at `runtime add` reaches an operator who is
	// present; a loop spawns unattended, so the durable record is the only
	// thing that can answer "who could have done this?" afterwards — which is
	// the question the repo-creation incident left unanswerable (task 308).
	if entry, why := store.UngatedOutwardGrant(append(append([]string{}, rt.Args...), extraArgs...)); entry != "" {
		fmt.Fprintf(ctx.Stderr, "warning: %s runs with %s — %s\n", childID, entry, why)
		_, _ = eventlog.Append(w, id.ID, model.EventFinding, t.Slug, "agent",
			fmt.Sprintf("spawned %s on runtime %s, whose allowlist includes %s — that child can write outward without passing dacli's consent gates", childID, rt.Name, entry))
	}
	fmt.Fprintf(ctx.Stderr, "spawning %s on %s for %03d-%s (run %s)\n", childID, rt.Name, t.Seq, t.Slug, clikit.Short(runID, 10))
	// Register the live process tree so `dacli agents`/`dacli kill` (a separate
	// invocation) can find and reap it while this spawn blocks here.
	var procWriteErr error
	var startedRec procmon.Record
	onStart := func(pid, pgid int) {
		startedRec = procmon.Record{
			RunID: runID, Child: childID, Task: t.ID, Role: roleName, Runtime: rt.Name,
			PID: pid, PGID: pgid, PIDStart: pidStart(pid), Started: time.Now(), Timeout: time.Duration(timeout) * time.Second, Claims: claims,
		}
		procWriteErr = procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), startedRec)
		if procWriteErr != nil {
			terminateRecordedTree(startedRec, 3*time.Second)
		}
	}
	transcriptPath := filepath.Join(runDir, "transcript.log")

	// --detach starts the child and returns immediately with a run-id, so an
	// orchestrator can launch many at once and block on them later with
	// `dacli wait` instead of hand-rolling shell backgrounding. The detached
	// child runs in its own process group (still visible to `dacli agents`,
	// killable by `dacli kill`); its outcome is finalized by `dacli wait`.
	if f.Bool("detach") {
		if _, _, derr := execRuntime(workDir, transcriptPath, rt, prompt, token, extraArgs, timeout, true, onStart); derr != nil {
			return fmt.Errorf("detached spawn failed to start: %w", derr)
		}
		if procWriteErr != nil {
			terminateRecordedTree(startedRec, 3*time.Second)
			return fmt.Errorf("record critical run artifact proc.txt: %w", procWriteErr)
		}
		if err := record.critical("outcome.md", fmt.Sprintf("outcome: running (detached)\nchild: %s\ntask: %s\n", childID, t.ID)); err != nil {
			terminateRecordedTree(startedRec, 3*time.Second)
			return err
		}
		if err := startRunWatchdog(w.Root, runID); err != nil {
			if rec, ok := readProcByRef(w, runID); ok {
				terminateRecordedTree(rec, 3*time.Second)
			}
			return fmt.Errorf("detached spawn watchdog: %w", err)
		}
		fmt.Fprintf(ctx.Stdout, "detached %s on %s for %03d-%s (run %s)\ntrack: dacli agents · block: dacli wait %s · transcript: %s\n",
			childID, rt.Name, t.Seq, t.Slug, clikit.Short(runID, 10), clikit.Short(runID, 10), transcriptPath)
		return nil
	}

	// cmd.Dir and worktreePreamble are the isolation SIGNAL sent to a
	// --worktree child: they are cooperative, not enforced (RUNTIMES.md § 8),
	// so a child that ignores its cwd — or a runtime whose own path allowlist
	// resolves back to the main checkout's absolute path, dacli-267 — can
	// still write there. preSpawnDirty snapshots the main checkout right
	// before the child runs so any NEW dirt it leaves behind can be told
	// apart from a human's own pre-existing uncommitted work and reverted,
	// rather than left for a separate `doctor` run to someday notice.
	var preSpawnDirty map[string]bool
	if isolatedWorktree {
		if before, derr := gitx.DirtyPaths(w.Root, ".dacli"); derr == nil {
			preSpawnDirty = make(map[string]bool, len(before))
			for _, p := range before {
				preSpawnDirty[p] = true
			}
		}
	}

	elapsed, timedOut, runErr := execRuntime(workDir, transcriptPath, rt, prompt, token, extraArgs, timeout, false, onStart)
	if procWriteErr != nil {
		return fmt.Errorf("record critical run artifact proc.txt: %w", procWriteErr)
	}
	if runErr != nil && !timedOut {
		if policyErr := recordProviderFailure(ctx, w, rt.Name, transcriptPath, runErr, record.bestEffort); policyErr != nil {
			return policyErr
		}
	}

	if isolatedWorktree {
		if leaked, rerr := reclaimMainCheckoutEscape(w.Root, preSpawnDirty); rerr != nil {
			fmt.Fprintf(ctx.Stderr, "warning: could not check main checkout for a worktree escape: %v\n", rerr)
		} else if len(leaked) > 0 {
			fmt.Fprintf(ctx.Stderr, "worktree escape: child %s wrote outside %s into the main checkout — reverted %d path(s): %s\n",
				childID, workDir, len(leaked), strings.Join(leaked, ", "))
			runErr = fmt.Errorf("child wrote outside its worktree into the main checkout (reverted): %s", strings.Join(leaked, ", "))
		}
	}
	if f.Bool("structured-review-result") && runErr == nil {
		if grant != model.GrantRO {
			runErr = clikit.Refusedf("independent review requires a read-only reviewer grant, got %s", grant)
		} else if err := materializeReviewOutput(w, t, childID, plan.Role, rt.Name, modelName, transcriptPath); err != nil {
			runErr = fmt.Errorf("structured review output: %w", err)
		}
	}

	// Evaluate against the fixed criterion: acceptance boxes, plus what the
	// child actually wrote to the workspace. Partial work survives a dead
	// child — that is the whole point of the workspace return channel.
	t2, _ := store.FindTask(w, taskRef)
	done, total := 0, 0
	if t2 != nil {
		for _, box := range t2.Acceptance() {
			total++
			if box.Done {
				done++
			}
		}
	}
	// Read the child's events from the shared root — where a worktree child now
	// writes them too (workspace.Find redirects), so the outcome reflects real
	// work instead of always reading 0.
	childEvents, _ := eventlog.List(w, eventlog.Query{Actor: childID})

	outcome := "ok"
	var handoff store.RootHandoff
	handoffRequired := false
	if store.RootHandoffRequested(w, runID) || (isolatedWorktree && len(plan.PlannedHandoffs) > 0) {
		failureClass := "filesystem_sandbox_refusal"
		if runErr != nil {
			failureClass = mutationFailureClass(runErr)
		}
		var handoffErr error
		handoff, handoffRequired, handoffErr = store.CaptureRootHandoff(w, runID, t.ID, childID, workDir, store.RootHandoffRequest{
			Schema: store.RootHandoffSchema, FailedOperation: "worker lifecycle publication",
			FailureClass: failureClass, Stderr: clikit.ErrStr(runErr),
			NextAction: "owner consumes the handoff after hash re-observation, reruns verification, then commits and publishes without changing the worker harness or grant",
		}, time.Now())
		if handoffErr != nil {
			return fmt.Errorf("capture root handoff: %w", handoffErr)
		}
	}
	switch {
	case handoffRequired:
		outcome = "handoff-required"
	case timedOut:
		outcome = "stalled"
	case runErr != nil && len(childEvents) > 0:
		outcome = "partial"
	case runErr != nil:
		outcome = "failed"
	}
	if err := record.critical("outcome.md", fmt.Sprintf("outcome: %s\nexit: %v\nelapsed: %s\nacceptance: %d/%d\nevents_by_child: %d\n",
		outcome, clikit.ErrStr(runErr), elapsed, done, total, len(childEvents))); err != nil {
		return err
	}
	if rec, ok := readProcByRef(w, runID); ok {
		if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, outcome); err != nil {
			return fmt.Errorf("record critical run artifact proc.txt: %w", err)
		}
	}

	fmt.Fprintf(ctx.Stdout, "run %s: %s in %s · child wrote %d event(s) · acceptance %d/%d\ntranscript: %s\n",
		clikit.Short(runID, 10), outcome, elapsed, len(childEvents), done, total, filepath.Join(runDir, "transcript.log"))
	if outcome == "failed" || outcome == "stalled" {
		if runErr != nil {
			return fmt.Errorf("child %s: %s (see %s): %w", childID, outcome, runDir, runErr)
		}
		return fmt.Errorf("child %s: %s (see %s)", childID, outcome, runDir)
	}
	if outcome == "handoff-required" {
		return clikit.Refusedf("handoff-required for run %s: %s; next: %s", clikit.Short(runID, 10), handoff.FailedOperation, handoff.NextAction)
	}
	return nil
}

// reclaimMainCheckoutEscape diffs the main checkout's dirty paths against
// preSpawnDirty (captured right before the child ran) and reverts whatever is
// new — a tracked file is restored with `git checkout --`, an untracked one
// (the common case: a brand-new file the child created by absolute path) is
// removed outright, since there is nothing to restore it to. Only the DELTA
// is touched, so a human's own pre-existing uncommitted work in the main
// checkout is never reverted. Returns the leaked paths (already undone) so
// the caller can name them.
//
// This is a backstop, not prevention: it cannot stop the write from
// happening, only make sure it does not survive the spawn that caused it. A
// child that commits directly into main's history (rather than dirtying its
// working tree) is not caught here — that needs the child to run `git
// commit` against a different repo path, a materially more deliberate
// action than the stray-file-write incident this guards against.
func reclaimMainCheckoutEscape(root string, preSpawnDirty map[string]bool) ([]string, error) {
	after, err := gitx.DirtyPaths(root, ".dacli")
	if err != nil {
		return nil, err
	}
	var leaked []string
	for _, p := range after {
		if !preSpawnDirty[p] {
			leaked = append(leaked, p)
		}
	}
	for _, p := range leaked {
		if _, err := gitx.Run(root, "checkout", "--", p); err != nil {
			// Not a tracked path checkout can restore (a brand-new file) —
			// remove it directly so the tree returns to its pre-spawn state.
			_ = os.Remove(filepath.Join(root, p))
		}
	}
	return leaked, nil
}

// worktreePreamble is the ISOLATED-WORKTREE section appended to a --worktree
// child's brief. It resolves two directions the agent would otherwise conflate:
//
//   - CODE and git live in the worktree, so edits land on THIS branch. That is
//     the sandbox-signal fix (the `dacli` binary is allowlisted at the main
//     checkout's path, so an agent can mistake main for its repo).
//   - dacli WORKSPACE STATE — identity, `task check`, notes, findings, and the
//     event crumb `dacli commit` writes — deliberately resolves to the shared
//     MAIN workspace, not this worktree's `.dacli` snapshot (workspace.Find
//     redirects a linked worktree via git's common dir; the snapshot is stale
//     the moment the branch was cut). Task 260: spawn must SAY this, because an
//     agent that assumes its `task check` lands on its branch is surprised when
//     the record shows up in the shared store — and may "fix" it by cd-ing to
//     main, which corrupts the shared tree.
func worktreePreamble(wtPath string) string {
	return fmt.Sprintf("\n\n## Your working directory (ISOLATED WORKTREE)\nYou are running in an isolated git worktree at:\n\n    %s\n\nEVERY file you read, create, or edit lives UNDER this directory — use paths relative to it. Do NOT edit any file by an absolute path outside it. In particular, the `dacli` binary may live in a DIFFERENT checkout (the main tree); editing code there clobbers the main tree and other agents. Your code edits, `git`, `go build`, and the commit `dacli commit` records all operate HERE, on THIS branch.\n\nBut dacli's WORKSPACE STATE is deliberately shared, not per-branch: your agent identity, `task check`, `note add`, findings, and the event crumb every `dacli commit` writes resolve to the MAIN workspace at the repo root — NOT this worktree's `.dacli` (a git snapshot that went stale the moment your branch was cut). That is intended and correct: it is how you see your own freshly-minted identity and how your reports reach the owner in one shared, append-safe store. So your CODE lands on your branch while your RECORD of the work lands in the shared store. Never `cd` to the main checkout to 'fix' this — that would commit code onto main's tree.\n", wtPath)
}

// blockedFileName is the break-glass channel (task 269). Every way a child
// reports — notes, findings, ask, task done — is a `dacli` invocation, so the
// ONE failure that breaks dacli itself (a broken binary, a rejected sandbox, a
// lost token) would silence every report and make the run look merely empty
// instead of blocked. A child writes this plain file into its run directory with
// a single ordinary file write and NO dacli, and `agents`/`wait` read it back to
// report the run as a distinct BLOCKED state. It lives in the run directory
// under the SHARED workspace (not the child's worktree), so it is read back the
// same way whether or not the child ran isolated, and it never lands on a branch.
const blockedFileName = "blocked.txt"

// readBlocked returns the child's blocked reason for a run, trimmed of trailing
// whitespace, or "" when the child never raised the channel. A read error
// (file absent — the common case) is deliberately reported as "not blocked".
func readBlocked(w *workspace.Workspace, runID string) string {
	raw, err := os.ReadFile(filepath.Join(w.RunDir(runID), blockedFileName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// firstLine returns s up to its first newline, trimmed — the one-line summary
// for a possibly-multiline blocked reason (the full text is kept whole in the
// file and in outcome.md; only the display is shortened).
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// blockedChannelPreamble is the last section of a spawned child's brief: it
// names the exact path of the break-glass BLOCKED channel and says when to use
// it. It is appended AFTER the worktree preamble so its single carve-out from
// the "never write an absolute path outside your worktree" rule — this one path
// is the shared record store, not a code tree — is the final word (task 269).
func blockedChannelPreamble(path string) string {
	return fmt.Sprintf("\n\n## If you cannot run dacli at all (BLOCKED)\nEvery channel above — `note add`, findings, `ask`, `task done` — is a `dacli` invocation. So the one failure that breaks `dacli` ITSELF (a broken binary, a sandbox that rejects it, a lost token) would silence every report you could otherwise make, and your run would read as merely empty rather than blocked.\n\nThere is one escape hatch that is NOT dacli: with a single ordinary file write and no command, write a plain text file to exactly this path:\n\n    %s\n\nFirst line: that you are blocked. Below it: WHY — what you ran and what it said. `dacli agents` and `dacli wait` read this file and surface your run as BLOCKED, a distinct state and not a normal completion, so a human is told. This is the ONE absolute path outside your working directory you may write — it is the shared record store, never a code tree, so it never touches your branch. Use it only as a last resort, when dacli genuinely will not run; if dacli works, report through it as usual.\n", path)
}

// externalRadius reports whether task t's brief sits in the blast radius of any
// external-origin artifact (RUNTIMES § 18), returning the distinct origins when
// it does. A single broad "external:" needle unions every external source's
// radius; ExposedBriefs answers whether this task is exposed. hasExternal
// distinguishes "clean, nothing external recorded" from "clean, not in radius".
// Shared by --advise (display) and the spawn-time taint gate (refusal) so the
// two never diverge.
func externalRadius(w *workspace.Workspace, t *store.Task) (origins []string, inRadius, hasExternal bool) {
	res, err := store.Taint(w, "external:")
	if err != nil || len(res.Hits) == 0 {
		return nil, false, false
	}
	for _, ref := range res.ExposedBriefs(w) {
		if ref == t.Slug {
			inRadius = true
			break
		}
	}
	if !inRadius {
		return nil, false, true
	}
	seen := map[string]bool{}
	for _, h := range res.Hits {
		if !seen[h.Origin] {
			seen[h.Origin] = true
			origins = append(origins, h.Origin)
		}
	}
	sort.Strings(origins)
	return origins, true, true
}

// printAdvisory is the body of `spawn --advise`: it reports what the log
// already knows at the spawn decision and returns, changing nothing. Two
// readouts, both reusing the existing machinery:
//
//   - Budget/sizing from the calibrated band. TOKENS are the F1 unit and are
//     PREFERRED whenever this band has any token-bearing sample: the suggested
//     token budget = median output-tokens/point × the task's Te once the band
//     has enough history (n≥10, D1's threshold); below that the figure is
//     PROVISIONAL and no firm number is printed. A band whose runs never
//     reported usage falls back to the honest wall-clock proxy (same n≥10 gate).
//   - Taint status: whether this task's brief sits in an external source's
//     blast radius, via store.Taint / TaintResult.ExposedBriefs.
func printAdvisory(ctx *clikit.Ctx, w *workspace.Workspace, t *store.Task, band store.Band, claims []string) {
	fmt.Fprintf(ctx.Stdout, "── advise · %03d-%s · band %s ──\n", t.Seq, t.Slug, band.String())
	if len(claims) == 0 {
		fmt.Fprintln(ctx.Stdout, "  claims: (none)")
	} else {
		fmt.Fprintf(ctx.Stdout, "  claims: %s\n", strings.Join(claims, ", "))
	}

	// One walk of the calibration samples backs both the token readout (preferred)
	// and the wall-clock fallback below, so --advise never re-walks RunsDir twice.
	samples := store.CalibrationSamples(w)

	if tokRatio, tn := store.MedianTokenRatio(samples, band); tn > 0 {
		// F1's measured token cost is the real unit — prefer it over wall-clock.
		if tp, ok := t.Estimate(); ok && tn >= 10 {
			fmt.Fprintf(ctx.Stdout,
				"  tokens: ~%.0f suggested (band ×%.0f median output-tokens/point on Te %.1f, n=%d)\n",
				tokRatio*tp.Expected(), tokRatio, tp.Expected(), tn)
			fmt.Fprintln(ctx.Stdout, "  (measured token cost, F1; cap this spawn with --max-tokens N)")
		} else if tn >= 10 {
			fmt.Fprintf(ctx.Stdout,
				"  tokens: band ×%.0f median output-tokens/point (n=%d) — estimate the task for a token figure\n",
				tokRatio, tn)
		} else {
			// Thin data: mark PROVISIONAL, print no firm suggested budget.
			fmt.Fprintf(ctx.Stdout,
				"  tokens: PROVISIONAL — band has token history but n=%d < 10 (median ×%.0f output-tokens/point); no calibrated number yet\n",
				tn, tokRatio)
		}
	} else {
		// Honest fallback: this band's runs never reported tokens, so the
		// wall-clock proxy is the best calibrated sizing available.
		var rs []float64
		for _, s := range samples {
			if s.Band == band {
				rs = append(rs, s.Ratio())
			}
		}
		if len(rs) >= 10 {
			med, p10, p90 := spm.Median(rs), percentile(rs, 10), percentile(rs, 90)
			if tp, ok := t.Estimate(); ok {
				te := tp.Expected()
				fmt.Fprintf(ctx.Stdout,
					"  budget: ~%.1f h suggested (p10–p90 %.1f–%.1f h) — band ×%.2f median hours/point on Te %.1f\n",
					med*te, p10*te, p90*te, med, te)
			} else {
				fmt.Fprintf(ctx.Stdout,
					"  budget: band ×%.2f median hours/point (p10–p90 ×%.2f–×%.2f) — estimate the task for an hour figure\n",
					med, p10, p90)
			}
			fmt.Fprintln(ctx.Stdout, "  (wall-clock proxy — this band's runtime reports no tokens; advisory — you still pass --budget)")
		} else {
			fmt.Fprintf(ctx.Stdout, "  budget: no band history yet (n=%d < 10) — no calibrated suggestion\n", len(rs))
		}
	}

	// External origins are the untrusted class (RUNTIMES § 18 cross-tree
	// injection); externalRadius unions every external source's radius and
	// answers whether this task's brief is in it. The same helper backs the
	// spawn-time gate, so --advise and the refusal never disagree.
	if origins, inRadius, hasExternal := externalRadius(w, t); inRadius {
		fmt.Fprintf(ctx.Stdout, "  taint: task %03d is in the blast radius of %s — audit before trusting this brief\n",
			t.Seq, strings.Join(origins, ", "))
	} else if hasExternal {
		fmt.Fprintln(ctx.Stdout, "  taint: clean (no external-origin artifact reaches this brief)")
	} else {
		fmt.Fprintln(ctx.Stdout, "  taint: clean (no external-origin artifacts recorded)")
	}
	fmt.Fprintln(ctx.Stdout, "── (preview only — no agent spawned; re-run without --advise to launch) ──")
}

// bandTokenBudget computes the expected token cost of spawning THIS band on task
// t from the band's measured output-tokens/point (F1): expected = median
// TokenRatio × the task's Te. It reads the SAME calibration samples the advisory
// displays via store.MedianTokenRatio, so the launch and `--advise` can never
// disagree on the estimate. n is the count of token-bearing samples in the
// band. ok is false when the band has no token history (a text runtime) or the
// task carries no three-point estimate. This helper predicts cost; enforcement
// is solely the adapter's TokenLimitFlag passed by tokenLimitArgs.
func bandTokenBudget(w *workspace.Workspace, t *store.Task, band store.Band) (expected float64, n int, ok bool) {
	ratio, n := store.MedianTokenRatio(store.CalibrationSamples(w), band)
	if n == 0 {
		return 0, 0, false
	}
	tp, est := t.Estimate()
	if !est {
		return 0, n, false
	}
	return ratio * tp.Expected(), n, true
}

// percentile returns the p-th (0..100) percentile of xs by linear
// interpolation. It is a DELIBERATE local copy of insight.percentile: the
// feature-slice isolation rule (cli/arch_test.go — slices never import each
// other) forbids execution importing insight, and this task's STRICT scope
// forbids hoisting the helper into spm/store — so a small in-slice copy is the
// only honest option. The math is identical; keep the two in sync.
func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	if len(s) == 1 {
		return s[0]
	}
	rank := p / 100 * float64(len(s)-1)
	lo := int(rank)
	if lo >= len(s)-1 {
		return s[len(s)-1]
	}
	return s[lo] + (rank-float64(lo))*(s[lo+1]-s[lo])
}

// cmdSupervise runs the RUNTIMES § 7 loop: spawn, evaluate against the
// acceptance criteria written before the work started, correct, repeat. It
// terminates because the criterion is external and turns are capped.
func cmdSupervise(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject(launchFlagsWith("max-turns", "pr", "review", "pr-number")...); err != nil {
		return err
	}
	taskRef := f.Get("task")
	if taskRef == "" {
		return clikit.Usagef("usage: dacli supervise --task <ref> [--runtime name] [--role r] [--max-turns N] [--grant ro|rw] [--model m] [--claim path,path] [--pr] [--review [--pr-number N]] [--budget N] [--max-tokens N [--allow-advisory-tokens]] [--timeout sec] [--cooperative|--allow-user-config] [--advise] [--force]")
	}
	// The SAME gated prologue `spawn` takes (launchGates). supervise sends this
	// brief to this runtime once per turn, so every gate that decides whether a
	// child may be launched at all decides it here too — see the shared-path
	// comment above resolveLaunch.
	plan, err := resolveLaunch(ctx, w, f, taskRef)
	if errors.Is(err, errAdviseOnly) {
		return nil // --advise previewed and stopped; nothing spawned (dacli 232)
	}
	if err != nil {
		return err
	}
	t, rt := plan.Task, plan.Runtime
	if err := validateReviewTarget(w, t, f); err != nil {
		return err
	}
	// workspace.Find redirects calls from a linked task checkout to the shared
	// root. Preserve the caller's registered task worktree nevertheless: a
	// correction launched from root after an audited recovery transfer has no
	// current worktree ownership record, so its governed commit is refused.
	workDir, isolatedWorktree, err := resolveSuperviseWorkDir(w, t, ctx.Cwd)
	if err != nil {
		return err
	}
	roleName, modelName, grant := plan.RoleName, plan.Model, plan.Grant
	claims, sandboxArgs := plan.Claims, plan.Sandbox
	budget, timeout := plan.Budget, plan.Timeout

	maxTurns := 3
	if n, err := f.Int("max-turns", 0); err != nil {
		return err
	} else if n > 0 {
		maxTurns = n
	}

	// ONE child identity across turns: ownership continuity is what lets the
	// child claim on turn 1 and check its own boxes on turn 2.
	childID, token, err := agentid.Spawn(w, id, roleName, grant)
	if err != nil {
		return err
	}
	// One child owns this task across turns; claim once (idempotent) so a
	// claim->completed span exists for calibrate to join (D1).
	claimTask(ctx, w, t, childID)

	unmetList := func() []string {
		cur, err := store.FindTask(w, taskRef)
		if err != nil {
			return nil
		}
		var unmet []string
		for _, box := range cur.Acceptance() {
			if !box.Done {
				unmet = append(unmet, box.Text)
			}
		}
		return unmet
	}

	for turn := 1; turn <= maxTurns; turn++ {
		b, err := brief.Assemble(w, taskRef, brief.Options{Budget: budget, Role: roleName})
		if err != nil {
			return err
		}
		suffix, perr := promptSuffix(w, f, t, childID, grant)
		if perr != nil {
			return perr
		}
		prompt := b.Render() + suffix
		if turn > 1 {
			// No session resume: each turn re-sends the brief plus the
			// correction (templated). Turn 3 is a mis-sizing signal, not
			// normal operation.
			correction, cerr := prompts.Render(w.PromptsDir(), "supervise_correction", map[string]any{
				"Turn": turn, "MaxTurns": maxTurns, "Unmet": unmetList(),
			})
			if cerr != nil {
				return cerr
			}
			prompt += "\n" + correction
		}
		if turn == 3 {
			fmt.Fprintf(ctx.Stderr, "note: turn 3 — under the small-task doctrine this usually means the task should be decomposed, not retried\n")
		}

		runID := ulid.New()
		runDir := w.RunDir(runID)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			return err
		}
		record := openRunRecord(runDir, ctx.Stderr)
		if len(plan.PlannedHandoffs) > 0 {
			if err := record.critical("planned-handoffs.txt", strings.Join(plan.PlannedHandoffs, "\n")+"\n"); err != nil {
				return err
			}
		}
		if isolatedWorktree {
			record.bestEffort("worktree.txt", workDir+"\n")
			prompt += worktreePreamble(workDir)
		}
		if grant == model.GrantRW {
			prompt += handoffChannelPreamble(runDir, plan.PlannedHandoffs)
		}
		if err := record.critical("brief.md", prompt); err != nil {
			return err
		}
		// Record role/model in the SAME OrDash canonical form cmdSpawn uses
		// (execution.go:373-374), so a supervise-completed task's run band
		// {role,model,rt} matches the {OrDash(role),OrDash(model),rt} band the
		// calibrate gate/advise compares against. Without this the band was
		// {"","",rt} and never equalled the gate's {"-","-",rt} sentinel form,
		// making every supervise actual dead weight for by-agent-band calibration.
		invocation := fmt.Sprintf("run: %s\nsupervise_turn: %d/%d\ntask: %s\nchild: %s\nrole: %s\nmodel: %s\nruntime: %s\nmax_tokens: %s\nmax_tokens_mode: %s\n",
			runID, turn, maxTurns, t.ID, childID, clikit.OrDash(roleName), clikit.OrDash(modelName), rt.Name, f.Get("max-tokens"), tokenLimitMode(plan))
		invocation += contextInvocation(plan.Role, plan.HasRole, plan.ContextOverride, plan.ContextSources)
		invocation += mutationPreflightInvocation(plan.MutationCapabilities)
		provenance, err := promptInvocation(w.PromptsDir(), prompt)
		if err != nil {
			return err
		}
		invocation += provenance
		if err := record.critical("invocation.txt", invocation); err != nil {
			return err
		}

		fmt.Fprintf(ctx.Stderr, "turn %d/%d: %s on %s\n", turn, maxTurns, childID, rt.Name)
		extraArgs := append(append([]string{}, sandboxArgs...), modelArgs(ctx, rt, modelName)...)
		extraArgs = append(extraArgs, tokenLimitArgs(plan)...)
		// Claims are PUBLISHED here, not merely checked in the prologue: the
		// overlap gate reads other agents' proc records, so a supervised run that
		// checked --claim without recording its own would take the mutual
		// exclusion one-way — every other agent would still be free to claim the
		// tree this one is editing.
		var procWriteErr error
		onStart := func(pid, pgid int) {
			rec := procmon.Record{
				RunID: runID, Child: childID, Task: t.ID, Role: roleName, Runtime: rt.Name,
				PID: pid, PGID: pgid, PIDStart: pidStart(pid), Started: time.Now(), Timeout: time.Duration(timeout) * time.Second, Claims: claims,
			}
			procWriteErr = procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), rec)
			if procWriteErr != nil {
				terminateRecordedTree(rec, 3*time.Second)
			}
		}
		elapsed, timedOut, runErr := execRuntime(workDir, filepath.Join(runDir, "transcript.log"), rt, prompt, token, extraArgs, timeout, false, onStart)
		if procWriteErr != nil {
			return fmt.Errorf("record critical run artifact proc.txt: %w", procWriteErr)
		}
		if runErr != nil && !timedOut {
			if policyErr := recordProviderFailure(ctx, w, rt.Name, filepath.Join(runDir, "transcript.log"), runErr, record.bestEffort); policyErr != nil {
				return policyErr
			}
		}

		// The supervisor owns the objects, so it applies the child's events
		// between turns — claims become ownership, findings become notes.
		if res, err := eventlog.Sync(w, id.ID, id.CanMutate); err == nil && res.Applied > 0 {
			fmt.Fprintf(ctx.Stderr, "  applied %d child event(s)\n", res.Applied)
		}

		unmet := unmetList()
		cur, _ := store.FindTask(w, taskRef)
		outcome := fmt.Sprintf("turn %d: %d unmet, elapsed %s", turn, len(unmet), elapsed)
		if err := record.critical("outcome.md", outcome+"\n"); err != nil {
			return err
		}
		if rec, ok := readProcByRef(w, runID); ok {
			if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, outcome); err != nil {
				return fmt.Errorf("record critical run artifact proc.txt: %w", err)
			}
		}

		if len(unmet) == 0 && cur != nil && cur.Status == model.StatusDone {
			fmt.Fprintf(ctx.Stdout, "accepted after %d turn(s): all acceptance criteria met and task done\n", turn)
			return nil
		}
		if timedOut {
			if runErr != nil {
				return fmt.Errorf("stalled: turn %d timed out after %ds (run %s): %w", turn, timeout, clikit.Short(runID, 10), runErr)
			}
			return fmt.Errorf("stalled: turn %d timed out after %ds (run %s)", turn, timeout, clikit.Short(runID, 10))
		}
		if runErr != nil {
			fmt.Fprintf(ctx.Stderr, "  turn %d exited non-zero (%v) — child events still count\n", turn, runErr)
		}
	}
	return fmt.Errorf("stalled after %d turns; unmet:\n  - %s\ndecompose the task or fix the criteria — do not simply re-run", maxTurns, strings.Join(unmetList(), "\n  - "))
}

// phaseGate is the answer to "don't start implementation while still in
// discovery": if the task's project is in a gated phase, a role whose kind
// the phase disallows is refused — you can't spawn an implementer into a
// research phase. Roles with no kind opt out; solo/untemplated projects are
// never gated.
func phaseGate(w *workspace.Workspace, t *store.Task, role team.Role) error {
	if role.Kind == "" {
		return nil
	}
	// A phase we could not READ is not a phase that permits everything. The
	// stage gates had this exact shape and were fixed to fail closed; this
	// preflight kept the old one, so a broken or unreadable project file let a
	// role of any kind be spawned into any phase — the gate reporting success
	// from a path that never ran the check (found by nilerr).
	ph, err := gates.PhaseFor(w, t.Project)
	if err != nil {
		return clikit.Refusedf("cannot read project %s's stage to check whether a %s role may act: %v — fix the project file, or pass --force if you have accounted for it", t.Project, role.Kind, err)
	}
	if !ph.Gated {
		return nil
	}
	if !ph.AllowsKind(role.Kind) {
		return clikit.Refusedf("project %s is in the %s phase; a %s role has no work here (allowed: %s). Advance the stage first (dacli stage advance %s), or use an allowed role",
			t.Project, ph.Name, role.Kind, strings.Join(ph.Allows, ", "), t.Project)
	}
	return nil
}

// seniorityGate enforces a role's MaxPoints: a junior role mechanically
// cannot take the hard migration. Unestimated tasks are refused too — a
// capped role takes only work whose size somebody stated.
//
// EXCEPT for a planner, whose OUTPUT is the size. Refusing one for a missing
// estimate asks a malformed question ("can you hold work of this size?" of the
// role that determines the size), and it deadlocked the loop: the review phase
// files unestimated tasks, sizeUnestimated spawns the capped `estimator` role
// to size them, this gate refused THAT spawn, and the capped implementer then
// refused the still-unsized task — every cycle, forever. The reporter watched
// fourteen consecutive no-progress cycles and their backlog grow 6 → 8 while
// done stayed at 27 (issue #430).
//
// Reproduced verbatim before fixing:
//
//	$ dacli spawn --task 344 --role estimator
//	dacli: role estimator takes only estimated tasks (max 2 points) — estimate 344-… first
//
// The over-cap refusal below still applies to a planner once an estimate
// exists, so the cap keeps meaning what it says.
func seniorityGate(role team.Role, t *store.Task) error {
	tp, ok := t.Estimate()
	te := 0.0
	if ok {
		te = tp.Expected()
	}
	capacity := team.TaskCapacity(role, te, ok)
	if capacity.Fits {
		return nil
	}
	if !ok {
		return clikit.Refusedf("role %s takes only estimated tasks (max %g points) — estimate %03d-%s first (a planner-kind role is exempt: sizing is its output)", role.Name, capacity.Limit, t.Seq, t.Slug)
	}
	return clikit.Refusedf("task %03d-%s is Te %.1f, above role %s's cap of %g — assign a heavier role, or decompose the task", t.Seq, t.Slug, te, role.Name, capacity.Limit)
}

// modelArgs routes a model tier onto the runtime. A runtime with no model
// flag makes role-level routing inoperative — announced, never ignored.
func modelArgs(ctx *clikit.Ctx, rt store.Runtime, modelName string) []string {
	if modelName == "" {
		return nil
	}
	if rt.ModelFlag == "" {
		fmt.Fprintf(ctx.Stderr, "warning: model %q requested but runtime %s declares no model_flag — running on the runtime's default\n", modelName, rt.Name)
		return nil
	}
	return []string{rt.ModelFlag, modelName}
}

// sandboxFor applies the § 8 rule: a read-only child needs a runtime that
// can enforce it. --cooperative downgrades EXPLICITLY and loudly.
func sandboxFor(ctx *clikit.Ctx, rt store.Runtime, grant model.Grant, cooperative bool) ([]string, error) {
	if grant != model.GrantRO {
		// The rw half of the same rule. An rw grant is a promise the child can
		// modify the workspace; on a runtime whose allowlist grants no write tool
		// (junior's cc: Read/Grep/Glob/LS + the dacli binary) that promise is a
		// lie — the child reads its brief, fails its first edit, and the run is
		// spent discovering it (dacli 250). Refuse at spawn, symmetric to the ro
		// refusal below, unless --cooperative accepts it out loud.
		if !store.RuntimeWritable(rt) && !cooperative {
			return nil, clikit.Refusedf("runtime %s grants no write tool (its --allowedTools list has no Edit/Write), so an rw child cannot modify the workspace and would burn the run finding out. Add a write-capable adapter — `dacli runtime add %s-rw --preset claude-code-rw` — and point the role at it, or pass --cooperative to spawn anyway", rt.Name, rt.Name)
		}
		return nil, nil
	}
	if store.RuntimeEnforcesRO(rt) {
		return rt.SandboxRO, nil
	}
	if !cooperative {
		state := rt.ROProbe
		if state == "" {
			state = store.RuntimeROUnknown
		}
		return nil, clikit.Refusedf("runtime %s cannot enforce read-only (sandbox probe: %s); spawning an unverified process labeled ro would be a lie. Run `dacli runtime doctor`, pass --cooperative to accept convention-only permissions, or use an rw grant", rt.Name, state)
	}
	if rt.ROProbe == store.RuntimeROFailed {
		fmt.Fprintf(ctx.Stderr, "warning: read-only is COOPERATIVE on %s — its sandbox probe failed, so no sandbox arguments are applied; the child can bypass dacli and you accepted this with --cooperative\n", rt.Name)
		return nil, nil
	}
	fmt.Fprintf(ctx.Stderr, "warning: read-only is COOPERATIVE on %s — declared sandbox arguments are unverified and applied best-effort; the child can bypass dacli and you accepted this with --cooperative\n", rt.Name)
	return rt.SandboxRO, nil
}

// recordProviderFailure is the adapter-neutral failure seam shared by spawn
// and supervise. Permanent, authentication, and policy outcomes are recorded
// as typed diagnostics but never open a fallback circuit.
func recordProviderFailure(ctx *clikit.Ctx, w *workspace.Workspace, runtimeName, transcriptPath string, runErr error, writeRun func(string, string)) error {
	exitCode := 1
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	return recordProviderOutcome(ctx.Stdout, w, runtimeName, transcriptPath, exitCode, writeRun)
}

func recordProviderOutcome(out io.Writer, w *workspace.Workspace, runtimeName, transcriptPath string, exitCode int, writeRun func(string, string)) error {
	raw, _ := os.ReadFile(transcriptPath)
	outcome := providerpolicy.Classify(exitCode, string(raw))
	writeRun("provider-outcome.txt", fmt.Sprintf("kind: %s\nreason: %s\nreset_after: %s\n", outcome.Kind, outcome.Reason, outcome.ResetAfter))
	if !outcome.Fallbackable() {
		return nil
	}
	cooldown := outcome.ResetAfter
	if cooldown <= 0 {
		if delay, ok := (providerpolicy.RetryPolicy{Base: time.Second, Max: time.Minute, Jitter: .2}).Delay(0, outcome); ok {
			cooldown = delay
		} else {
			// Hard quota exhaustion has no useful immediate retry.
			cooldown = time.Hour
		}
	}
	limits := store.LoadRuntimeLimits(w)
	if _, err := limits.Record(runtimeName, outcome, cooldown); err != nil {
		return err
	}
	return limits.Report(out, providerpolicy.Transition{Source: runtimeName, Reason: outcome.Reason, Cooldown: cooldown})
}

// execRuntime launches one child turn. Env is allowlisted by NAME — the
// child gets the token plus exactly what the adapter declares, never the
// parent's full environment.
//
// A private guardian is placed in its own process GROUP (Setpgid), then starts
// the runtime in that group and remains its authenticated leader until every
// descendant drains. This preserves task 177's dead-runtime descendants without
// trusting a recycled numeric PGID (task 369). onStart records the guardian's
// durable pid/start identity for separate dacli invocations.
//
// The child's stdout+stderr stream to transcriptPath (a real file), so a
// DETACHED child's output persists after this parent process exits: the child
// keeps its own inherited fd and the parent closes its copy. detach=true starts
// the child in its own process group, releases it, and returns immediately with
// no deadline (enforce timeouts via `dacli kill` or a watchdog); the foreground
// path keeps the context deadline and group-kill-on-timeout.
func execRuntime(dir, transcriptPath string, rt store.Runtime, prompt, token string, extraArgs []string, timeoutSec int, detach bool, onStart func(pid, pgid int)) (elapsed time.Duration, timedOut bool, err error) {
	argv := append([]string{}, rt.GlobalArgs...)
	argv = append(argv, rt.Args...)
	argv = append(argv, extraArgs...)
	// F1: opt-in usage capture. Only when the adapter sets usage_format do we
	// ask the child to emit a machine-readable event stream; an empty
	// UsageFormat leaves argv (and thus a text runtime) exactly as it was. The
	// claude CLI requires --verbose alongside stream-json under --print.
	streamJSON := rt.UsageFormat == "stream-json" || rt.UsageFormat == "codex-jsonl" || rt.UsageFormat == "gemini-stream-json" || rt.UsageFormat == "copilot-json"
	switch rt.UsageFormat {
	case "stream-json":
		argv = append(argv, "--output-format", "stream-json", "--verbose")
	case "gemini-stream-json":
		argv = append(argv, "--output-format", "stream-json")
	case "copilot-json":
		argv = append(argv, "--output-format", "json")
	}
	if rt.Mode == "arg" {
		if rt.Flag != "" {
			argv = append(argv, rt.Flag)
		}
		argv = append(argv, prompt)
	}
	// The denylist is enforced HERE, at the point of use, not only in
	// `runtime add`. Gating just the writer protected one door into a file:
	// a runtime .md hand-edited by an rw agent, written by an older dacli,
	// restored from git, or copied in had its env_passthrough honored
	// verbatim — handing a child the operator's API keys. This is the read
	// every spawn actually makes, so it is the only place the rule cannot be
	// walked around.
	env := []string{agentid.EnvVar + "=" + token}
	for _, name := range rt.Env {
		if bad := deniedEnvPassthrough([]string{name}); bad != "" {
			return 0, false, clikit.Refusedf("runtime %s declares env_passthrough %s, which carries a credential — remove it from %s; a child must never inherit the operator's keys",
				rt.Name, bad, rt.Name)
		}
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}
	var sink *os.File
	if transcriptPath != "" {
		sink, err = os.Create(transcriptPath)
		if err != nil {
			return 0, false, fmt.Errorf("create transcript %q: %w", transcriptPath, err)
		}
	}
	start := time.Now()
	runtimePath, err := exec.LookPath(rt.Binary)
	if err != nil {
		missing := exec.Command(rt.Binary)
		missing.Dir = dir
		return 0, false, commandresult.NewExternalError(missing, commandresult.RunOptions{
			Operation: "runtime " + rt.Name + " launch", WorkspaceRoot: dir,
		}, nil, nil, err, false)
	}
	exe, err := os.Executable()
	if err != nil {
		return 0, false, fmt.Errorf("resolve dacli guardian: %w", err)
	}
	guardianArgv := []string{"__run-guardian"}
	if transcriptPath != "" {
		// Detached launches outlive this process, so their runtime exit status
		// cannot be returned through execRuntime. The guardian persists it beside
		// the transcript for `dacli wait` to classify (issue #550).
		guardianArgv = append(guardianArgv, "--exit-file", filepath.Join(filepath.Dir(transcriptPath), "runtime-exit.txt"))
	}
	guardianArgv = append(guardianArgv, runtimePath)
	guardianArgv = append(guardianArgv, argv...)

	if detach {
		// Detached: no CommandContext (its deadline would fire on the parent's
		// exit and kill the child). New process group so the tree stays killable
		// and survives this process as its own group; Release() hands it off.
		cmd := exec.Command(exe, guardianArgv...)
		cmd.Dir = dir
		cmd.Env = env
		setNewProcessGroup(cmd)
		if sink != nil {
			cmd.Stdout, cmd.Stderr = sink, sink
		}
		if rt.Mode == "stdin" {
			// A non-*os.File Stdin (e.g. strings.Reader) makes os/exec spawn a
			// parent-side goroutine to copy prompt→pipe, drained only by Wait().
			// Detach calls Release() and returns WITHOUT Wait(), so the parent
			// exits and that goroutine dies mid-copy — a prompt larger than the
			// ~64KB pipe buffer (briefs routinely are) is truncated or lost. Back
			// the child's stdin with a real *os.File instead: its fd is inherited
			// directly at exec, so the child reads the whole prompt with no parent
			// involvement. The unlinked temp file's inode survives via the child's
			// open fd until the child finishes reading.
			tf, terr := os.CreateTemp("", "dacli-stdin-*")
			if terr != nil {
				return 0, false, fmt.Errorf("detached stdin prompt: %w", terr)
			}
			defer func() { _ = tf.Close(); _ = os.Remove(tf.Name()) }()
			if _, werr := tf.WriteString(prompt); werr != nil {
				return 0, false, fmt.Errorf("detached stdin prompt: %w", werr)
			}
			if _, serr := tf.Seek(0, io.SeekStart); serr != nil {
				return 0, false, fmt.Errorf("detached stdin prompt: %w", serr)
			}
			cmd.Stdin = tf
		}
		serr := cmd.Start()
		if sink != nil {
			_ = sink.Close() // the child kept its own dup of the fd
		}
		if serr != nil {
			return 0, false, commandresult.NewExternalError(cmd, commandresult.RunOptions{
				Operation: "runtime guardian start", WorkspaceRoot: dir,
			}, nil, nil, serr, false)
		}
		if onStart != nil {
			onStart(cmd.Process.Pid, cmd.Process.Pid)
		}
		// Reap the child in the background instead of Release()ing it (dacli
		// 217). Release drops our handle without ever waiting, so the child
		// becomes a ZOMBIE the moment it exits — harmless under `dacli spawn`
		// (that parent exits immediately and init reaps the child), but a
		// long-lived parent — `dacli mcp serve`, or any in-process driver —
		// keeps the corpse in the process table for its whole lifetime. A
		// zombie answers signal-0, so procmon would report the finished agent
		// live forever: phantom rows in `dacli agents`, `dacli wait` blocking
		// to its timeout, KillTree escalating to SIGKILL against a corpse, and
		// the PID pinned so no other agent can be recorded under it.
		//
		// The wait runs in a goroutine so detach stays non-blocking and the
		// child still outlives us: if this process exits first the goroutine
		// simply dies and init inherits the child, exactly as before. Liveness
		// is zombie-aware too (procmon.Alive) — belt and braces, because a
		// child of some OTHER long-lived parent is not ours to reap.
		go func() { _, _ = cmd.Process.Wait() }()
		return 0, false, nil
	}

	interruptCtx, stopInterrupt := signal.NotifyContext(context.Background(), interruptSignals()...)
	defer stopInterrupt()
	cctx, cancel := context.WithTimeout(interruptCtx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, exe, guardianArgv...)
	cmd.Dir = dir
	cmd.Env = env
	// New process group: the child becomes group leader (pgid == its pid), and
	// every subprocess it forks inherits the group unless it detaches.
	setNewProcessGroup(cmd)
	// On timeout/cancel, kill the whole GROUP. The default CommandContext
	// cancel kills only the leader — which would orphan the children the agent
	// spawned, exactly the runaway leak we are preventing.
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			_ = killProcessGroup(cmd.Process.Pid)
		}
		return nil
	}
	// Bound how long Wait blocks on output a grandchild may still hold open
	// after the group was killed, so a hung tree can't wedge dacli.
	cmd.WaitDelay = 5 * time.Second
	providerCmd := exec.Command(runtimePath)
	providerCmd.Dir = dir
	var stdoutDiagnostic, stderrDiagnostic runtimeDiagnosticTail
	wrapRuntimeFailure := func(cause error, timeout bool) error {
		if cause == nil {
			return nil
		}
		return commandresult.NewExternalError(providerCmd, commandresult.RunOptions{
			Operation: "runtime " + rt.Name + " launch", WorkspaceRoot: dir,
		}, stdoutDiagnostic.Bytes(), stderrDiagnostic.Bytes(), cause, timeout)
	}

	// stream-json capture: read the child's stdout through a pipe, tee a
	// human-readable rendering into the transcript (so logs -f / --tail keep
	// working) and remember the final usage event. Text runtimes keep the raw
	// stdout+stderr → sink wiring exactly as before.
	var streamPipe io.ReadCloser
	if streamJSON && sink != nil {
		streamPipe, _ = cmd.StdoutPipe()
		cmd.Stderr = io.MultiWriter(sink, &stderrDiagnostic)
		defer func() { _ = sink.Close() }()
	} else if sink != nil {
		cmd.Stdout = io.MultiWriter(sink, &stdoutDiagnostic)
		cmd.Stderr = io.MultiWriter(sink, &stderrDiagnostic)
		defer func() { _ = sink.Close() }()
	} else {
		cmd.Stdout, cmd.Stderr = &stdoutDiagnostic, &stderrDiagnostic
	}
	if rt.Mode == "stdin" {
		cmd.Stdin = strings.NewReader(prompt)
	}
	if serr := cmd.Start(); serr != nil {
		return time.Since(start).Round(time.Millisecond), false, commandresult.NewExternalError(cmd, commandresult.RunOptions{
			Operation: "runtime guardian start", WorkspaceRoot: dir,
		}, stdoutDiagnostic.Bytes(), stderrDiagnostic.Bytes(), serr, false)
	}
	if onStart != nil {
		onStart(cmd.Process.Pid, cmd.Process.Pid) // pgid == leader pid under Setpgid
	}
	if streamPipe != nil {
		// Must drain the pipe fully before Wait (os/exec closes it on exit).
		u := teeStructuredJSON(io.TeeReader(streamPipe, &stdoutDiagnostic), sink, rt.UsageFormat)
		err = cmd.Wait()
		if u.found {
			writeUsage(filepath.Dir(transcriptPath), u)
		} else if u.scanErr != nil {
			// The stream ended before the result event: usage was lost. Make that
			// visible in the transcript instead of falling back to the wall-clock
			// proxy as if this were a plain text runtime.
			fmt.Fprintf(sink, "[dacli: usage capture incomplete — %v]\n", u.scanErr)
		}
		timedOut = cctx.Err() == context.DeadlineExceeded
		return time.Since(start).Round(time.Millisecond), timedOut, wrapRuntimeFailure(err, timedOut)
	}
	err = cmd.Wait()
	timedOut = cctx.Err() == context.DeadlineExceeded
	return time.Since(start).Round(time.Millisecond), timedOut, wrapRuntimeFailure(err, timedOut)
}

// runtimeDiagnosticTail bounds custom streaming captures before they reach the
// shared commandresult redaction/classification policy. Runtime transcripts can
// be arbitrarily large; diagnostics retain only their actionable end.
type runtimeDiagnosticTail struct{ b []byte }

func (t *runtimeDiagnosticTail) Write(p []byte) (int, error) {
	const limit = 8 << 10
	written := len(p)
	if len(p) >= limit {
		t.b = append(t.b[:0], p[len(p)-limit:]...)
		return written, nil
	}
	if overflow := len(t.b) + len(p) - limit; overflow > 0 {
		copy(t.b, t.b[overflow:])
		t.b = t.b[:len(t.b)-overflow]
	}
	t.b = append(t.b, p...)
	return written, nil
}

func (t *runtimeDiagnosticTail) Bytes() []byte { return append([]byte(nil), t.b...) }

// streamUsage is the final `result` event's accounting from a stream-json run.
type streamUsage struct {
	InputTokens  int
	OutputTokens int
	NumTurns     int
	CostUSD      float64
	SessionID    string
	FinalMessage string
	ExitOutcome  string
	found        bool
	// scanErr is a non-EOF read error (or over-long line) that ended the stream
	// BEFORE the terminating `result` event was seen. The result event carries
	// the ONLY usage numbers and arrives last, so an error mid-stream silently
	// loses token capture; callers surface scanErr instead of mistaking it for a
	// clean text-runtime EOF.
	scanErr error
}

type codexEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Item     struct {
		Type string `json:"type"`
		Text string `json:"text"`
		Name string `json:"name"`
	} `json:"item"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func renderCodexLine(line []byte, prior streamUsage) (string, streamUsage) {
	var ev codexEvent
	if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil {
		return string(bytes.TrimSpace(line)), prior
	}
	switch ev.Type {
	case "thread.started":
		prior.SessionID = ev.ThreadID
	case "item.completed":
		if ev.Item.Type == "agent_message" {
			prior.FinalMessage = strings.TrimSpace(ev.Item.Text)
			return prior.FinalMessage, prior
		}
		if ev.Item.Type != "" {
			return "[item: " + ev.Item.Type + "]", prior
		}
	case "turn.completed":
		prior.InputTokens, prior.OutputTokens, prior.ExitOutcome, prior.found = ev.Usage.InputTokens, ev.Usage.OutputTokens, "completed", true
	case "turn.failed":
		prior.ExitOutcome, prior.found = "failed", true
	}
	return "", prior
}

func teeStructuredJSON(r io.Reader, out io.Writer, format string) streamUsage {
	if format == "gemini-stream-json" {
		return teeGeminiStreamJSON(r, out)
	}
	if format != "codex-jsonl" {
		return teeStreamJSON(r, out)
	}
	var u streamUsage
	br := bufio.NewReaderSize(r, 64*1024)
	for {
		line, err := br.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var text string
			text, u = renderCodexLine(line, u)
			if text != "" {
				fmt.Fprintln(out, text)
			}
		}
		if err != nil {
			if err != io.EOF {
				u.scanErr = err
			}
			break
		}
	}
	return u
}

type geminiEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	Stats     struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"stats"`
}

func teeGeminiStreamJSON(r io.Reader, out io.Writer) streamUsage {
	var u streamUsage
	var final strings.Builder
	br := bufio.NewReaderSize(r, 64*1024)
	for {
		line, err := br.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var ev geminiEvent
			if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil {
				fmt.Fprintln(out, string(bytes.TrimSpace(line)))
			} else {
				switch ev.Type {
				case "init":
					u.SessionID = ev.SessionID
				case "message":
					if ev.Role == "assistant" {
						fmt.Fprint(out, ev.Content)
						final.WriteString(ev.Content)
					}
				case "result":
					u.InputTokens, u.OutputTokens = ev.Stats.InputTokens, ev.Stats.OutputTokens
					u.FinalMessage = strings.TrimSpace(final.String())
					u.ExitOutcome = map[bool]string{true: "completed", false: "failed"}[ev.Status == "success"]
					u.found = true
				}
			}
		}
		if err != nil {
			if err != io.EOF {
				u.scanErr = err
			}
			break
		}
	}
	return u
}

// streamEvent is the subset of a `claude --output-format stream-json` event we
// read: assistant content (for the readable rendering) and the result event's
// usage accounting.
type streamEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	IsError   bool   `json:"is_error"`
	Message   struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
			Name string `json:"name"`
		} `json:"content"`
	} `json:"message"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	NumTurns int     `json:"num_turns"`
	CostUSD  float64 `json:"total_cost_usd"`
}

// renderStreamLine turns one stream-json line into its human-readable rendering
// (assistant text and [tool: X] markers) and, when it is the terminating
// `result` event, its usage. A line that is not a JSON event is returned
// verbatim so nothing the child emits is ever dropped. text is "" for events
// with no human-facing content (system/result/empty), letting callers skip
// them. This is the single shared decoder for both the live tee and the
// render-on-read transcript readers, so foreground and detached runs render
// identically.
func renderStreamLine(line []byte) (text string, usage streamUsage) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return "", streamUsage{}
	}
	if trimmed[0] != '{' { // fast path: not an event object — pass through
		return string(trimmed), streamUsage{}
	}
	var ev streamEvent
	if err := json.Unmarshal(trimmed, &ev); err != nil {
		return string(trimmed), streamUsage{} // not an event — verbatim
	}
	switch ev.Type {
	case "assistant":
		var b strings.Builder
		for _, c := range ev.Message.Content {
			switch c.Type {
			case "text":
				if s := strings.TrimSpace(c.Text); s != "" {
					b.WriteString(s)
					b.WriteByte('\n')
				}
			case "tool_use":
				fmt.Fprintf(&b, "[tool: %s]\n", c.Name)
			}
		}
		return strings.TrimRight(b.String(), "\n"), streamUsage{}
	case "result":
		return "", streamUsage{
			InputTokens:  ev.Usage.InputTokens,
			OutputTokens: ev.Usage.OutputTokens,
			NumTurns:     ev.NumTurns,
			CostUSD:      ev.CostUSD,
			SessionID:    ev.SessionID,
			FinalMessage: strings.TrimSpace(ev.Result),
			ExitOutcome:  map[bool]string{false: "completed", true: "failed"}[ev.IsError],
			found:        true,
		}
	}
	return "", streamUsage{}
}

// teeStreamJSON reads a stream-json event stream from r, writes a human-readable
// rendering to out so the transcript stays as legible as a text runtime's, and
// returns the usage carried by the terminating `result` event. It uses a
// bufio.Reader (not a Scanner) so a single very large event line cannot exceed a
// buffer cap and abort the stream before the result event — the failure that
// silently lost usage. Any non-EOF read error is reported in the returned
// streamUsage.scanErr rather than swallowed.
func teeStreamJSON(r io.Reader, out io.Writer) streamUsage {
	var u streamUsage
	br := bufio.NewReaderSize(r, 64*1024)
	for {
		line, err := br.ReadBytes('\n') // no length cap: over-long lines grow, never truncate
		if len(bytes.TrimSpace(line)) > 0 {
			text, usage := renderStreamLine(line)
			if text != "" {
				fmt.Fprintln(out, text)
			}
			if usage.found {
				u = usage
			}
		}
		if err != nil {
			if err != io.EOF {
				u.scanErr = err
			}
			break
		}
	}
	return u
}

// writeUsage records the captured token accounting into the run record so
// calibration can read it back (store.CalibrationSamples). Best-effort: a
// missing usage.txt just means calibration falls back to the wall-clock proxy.
func writeUsage(runDir string, u streamUsage) {
	record := openRunRecord(runDir, nil)
	body := fmt.Sprintf("output_tokens: %d\ninput_tokens: %d\nnum_turns: %d\ncost_usd: %.6f\n",
		u.OutputTokens, u.InputTokens, u.NumTurns, u.CostUSD)
	record.bestEffort("usage.txt", body)
	if u.SessionID != "" || u.FinalMessage != "" || u.ExitOutcome != "" {
		result := fmt.Sprintf("session_id: %s\nexit_outcome: %s\nfinal_message: %s\n", u.SessionID, u.ExitOutcome, u.FinalMessage)
		record.bestEffort("result.txt", result)
	}
}

// promptSuffix assembles everything appended after the brief: the reporting
// protocol, git discipline for writers, review discipline for reviewers.
// All of it lives in the prompt registry, none of it in Fprintf chains.
func promptSuffix(w *workspace.Workspace, f *clikit.Flags, t *store.Task, childID string, grant model.Grant) (string, error) {
	contract, err := prompts.AutonomousContract(w.PromptsDir())
	if err != nil {
		return "", fmt.Errorf("autonomous delivery contract: %w", err)
	}
	out, err := protocolPreamble(w, childID, grant, t)
	if err != nil {
		return "", err
	}
	if grant == model.GrantRW {
		exe, exeErr := os.Executable()
		if exeErr != nil {
			exe = "dacli"
		}
		git, err := prompts.Render(w.PromptsDir(), "git_workflow", map[string]any{
			// Commands that mutate task state must use the globally stable ID.
			// A sequence is only project-local, so emitting it here stranded
			// workers as soon as another project allocated the same number
			// (issue #636).
			"Ref":    t.ID,
			"Title":  t.Title,
			"Branch": fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug),
			"PR":     f.Bool("pr"),
			"Exe":    exe,
			// The child's format/build/test advice comes from the project's
			// own recorded stack (dacli 192) — this prompt used to order every
			// writer to gofmt its .go files, which is nonsense in a Python
			// project. A project with nothing recorded yields the zero Stack,
			// which the template treats as "unchanged from before".
			"Stack": projectStack(w, t.Project),
		})
		if err != nil {
			return "", err
		}
		out += "\n" + git
	}
	if f.Bool("review") {
		exe, exeErr := os.Executable()
		if exeErr != nil {
			exe = "dacli"
		}
		review, err := prompts.Render(w.PromptsDir(), "review_workflow", map[string]any{
			"Task":             t.ID,
			"Search":           fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug),
			"Base":             reviewBase(w, t),
			"PRRef":            f.Get("pr-number"),
			"PR":               f.Bool("pr"),
			"Exe":              exe,
			"StructuredReview": f.Bool("structured-review-result"),
		})
		if err != nil {
			return "", err
		}
		out += "\n" + review
	}
	return "\n" + contract.Text + out, nil
}

// reviewBase matches local review instructions to the task project's landing
// target. An explicitly configured base wins; legacy projects use the
// repository's actual trunk (including master or a renamed origin HEAD).
func reviewBase(w *workspace.Workspace, t *store.Task) string {
	if p, err := store.LoadProject(w, t.Project); err == nil && p.Landing.Base != "" {
		return p.Landing.Base
	}
	if trunk := store.TrunkBranch(w); trunk != "" {
		return trunk
	}
	return "main"
}

// promptInvocation records both levels of prompt identity. The contract hash
// changes when shared semantics change; prompt_hash also captures the task
// brief, role prose, worktree path, and last-mile recovery channel actually
// delivered to this invocation (issue #707).
func promptInvocation(overrideDir, prompt string) (string, error) {
	contract, err := prompts.AutonomousContract(overrideDir)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("prompt_schema: %s\nprompt_version: %s\ncontract_hash: sha256:%s\nprompt_hash: sha256:%s\n",
		contract.Schema, contract.Version, contract.Hash, prompts.DeliveredHash(prompt)), nil
}

// projectStack loads a task's project and reads back the stack `dacli new`
// recorded on it (dacli 192). Every failure — no project, unreadable doc, a
// project written before stacks were recorded — collapses to the zero Stack on
// purpose: a spawn must never fail because a prompt wanted to know the language,
// and the zero value is exactly the pre-192 behavior.
func projectStack(w *workspace.Workspace, slug string) prompts.Stack {
	if slug == "" {
		return prompts.Stack{}
	}
	p, err := store.LoadProject(w, slug)
	if err != nil {
		return prompts.Stack{}
	}
	return prompts.StackFromProject(p.Doc)
}

// protocolPreamble tells a spawned child HOW to report. Without it, a real
// headless child does the work and prints text into the void — work not
// written to the workspace does not exist.
func protocolPreamble(w *workspace.Workspace, childID string, grant model.Grant, t *store.Task) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		exe = "dacli"
	}
	out, err := prompts.Render(w.PromptsDir(), "protocol_preamble", map[string]any{
		"ChildID": childID,
		"Grant":   string(grant),
		"Ref":     t.ID,
		"Slug":    t.Slug,
		"Project": t.Project,
		"Exe":     exe,
		"RW":      grant == model.GrantRW,
	})
	if err != nil {
		// A broken override must not silently mute the protocol — the child
		// would work into the void. Fail the spawn instead.
		return "", fmt.Errorf("protocol_preamble template: %w (fix or remove the override in .dacli/prompts/)", err)
	}
	return "\n" + out, nil
}

func cmdRunsList(ctx *clikit.Ctx, args []string) error {
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
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		// No runs yet is normal and lists nothing. An unreadable runs directory
		// is a different fact, and reporting both as "no runs" hid the second
		// entirely. This function CAN return an error, unlike its two siblings
		// that cannot (see dacli 337).
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("cannot read the runs directory at %s: %w", w.RunsDir(), err)
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names))) // ULIDs: newest first
	for _, n := range names {
		line := "(no outcome recorded)"
		if raw, err := os.ReadFile(filepath.Join(w.RunDir(n), "outcome.md")); err == nil {
			line = strings.ReplaceAll(strings.TrimSpace(string(raw)), "\n", " · ")
		}
		fmt.Fprintf(ctx.Stdout, "%s  %s\n", clikit.Short(n, 10), line)
	}
	return nil
}

func cmdRunsShow(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject(); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli runs show <run-id-prefix>")
	}
	entries, _ := os.ReadDir(w.RunsDir())
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), f.Pos[0]) {
			continue
		}
		for _, name := range []string{"invocation.txt", "outcome.md", store.RootHandoffFile, "brief.md", "transcript.log", "diagnostics.txt"} {
			if raw, err := os.ReadFile(filepath.Join(w.RunDir(e.Name()), name)); err == nil {
				fmt.Fprintf(ctx.Stdout, "=== %s ===\n%s\n", name, strings.TrimSpace(string(raw)))
			}
		}
		// First match wins, DELIBERATELY, and this return is a success rather
		// than a verdict — the distinction that makes it unlike the two landing
		// bugs the candidate-loop sweep found (task 363), where an early return
		// of a NEGATIVE result made every later candidate unreachable. Run ids
		// are ULIDs, so a prefix long enough to be typed is effectively unique,
		// and entries are read in sorted order, so the choice is deterministic.
		//
		// It does mean an ambiguous prefix shows one run without saying so,
		// where FindTask refuses an ambiguous task ref outright. That
		// inconsistency is recorded as a finding rather than changed here:
		// tightening it is a behaviour change to a read-only command, not part
		// of the sweep.
		return nil
	}
	return store.ErrNotFound{Ref: "run " + f.Pos[0]}
}

func cmdRunsPrune(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("keep"); err != nil {
		return err
	}
	keep := 20
	if n, err := f.Int("keep", 0); err != nil {
		return err
	} else if n > 0 {
		keep = n
	}
	entries, _ := os.ReadDir(w.RunsDir())
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // oldest first
	// Everything outside the newest `keep` is a prune CANDIDATE — but a
	// candidate whose process is still running is skipped (dacli 208).
	// proc.txt, the transcript and the usage file are the only handles dacli
	// has on a live agent: `agents`, `wait` and `kill` all read them back from
	// disk. RemoveAll on a still-executing run orphans that agent — it keeps
	// burning tokens with nothing able to observe or stop it. Retention is a
	// disk-space policy; it does not get to blind us to a running process.
	// Skips are reported, not silently absorbed, so an operator who expected
	// `--keep 5` and got 7 directories knows exactly which two ran long.
	pruned, skipped := 0, 0
	for _, n := range names[:max(0, len(names)-keep)] {
		if rec, rerr := procmon.ReadRecord(filepath.Join(w.RunDir(n), "proc.txt")); rerr == nil && runStillLive(rec) {
			skipped++
			fmt.Fprintf(ctx.Stdout, "kept %s: %s still live (pid %d, group %d) — pruning it would orphan a running agent\n",
				clikit.Short(n, 10), clikit.OrDash(rec.Child), rec.PID, rec.PGID)
			continue
		}
		if err := os.RemoveAll(w.RunDir(n)); err != nil {
			return err
		}
		pruned++
	}
	fmt.Fprintf(ctx.Stdout, "pruned %d run(s), kept %d\n", pruned, len(names)-pruned)
	if skipped > 0 {
		fmt.Fprintf(ctx.Stdout, "%d live run(s) kept beyond --keep %d; re-run after they finish\n", skipped, keep)
	}
	return nil
}

// lifecycleNow is the single observation clock for every run-liveness reader.
// Keeping it as a seam makes the startup/transcript grace boundary testable
// without asking a loaded scheduler to finish several commands inside a small
// wall-clock margin (issue #896).
var lifecycleNow = time.Now

// cmdAgents lists agents whose process tree is still alive, with the RAM/CPU
// (and GPU where measurable) the whole group is holding right now, plus each
// agent's honest activity state (agentstate.Derive — thinking/acting/waiting/
// stalled/blocked/silent) so RAM and uptime alone never have to answer "is it
// still working?" A run's proc.txt is written at spawn; liveness is probed
// live, so an exited agent simply doesn't appear — the list is
// runaways-included, ghosts-excluded. During the bounded registration window,
// fresh transcript activity is also treated as live by runLifecycleLive.
func cmdAgents(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("max-rss", "max-runtime", "reap", "tail", "project"); err != nil {
		return err
	}
	project := f.Get("project")
	if project != "" && !workspace.SafeSegment(project) {
		return clikit.Usagef("--project requires a valid project slug")
	}
	if ctx.JSON || project != "" {
		if f.Bool("reap") || f.Get("max-rss") != "" || f.Get("max-runtime") != "" || f.Bool("tail") {
			return clikit.Usagef("--project/--json is a read-only progress view and cannot be combined with --reap, resource limits, or --tail")
		}
		return renderAgentProgress(ctx, w, project, time.Now())
	}
	// Listing agents is a read; --reap KILLS whole process trees, which `kill`
	// has always required rw for. The gate is here rather than on the command
	// table because the two live under one path: a read-only agent must keep
	// its view of the swarm while never being able to end a sibling's run.
	if f.Bool("reap") {
		if err := clikit.RequireRW(id, "reaping an agent (--reap)"); err != nil {
			return err
		}
	}
	// Optional budgets: an agent over either limit is a runaway. --reap kills
	// it (whole tree); without --reap it is only flagged, so you can look first.
	maxRSS := parseBytes(f.Get("max-rss"))           // e.g. 2G, 500M; 0 = no limit
	maxRun := parseDurationArg(f.Get("max-runtime")) // e.g. 15m, 900; 0 = no limit
	reap := f.Bool("reap")
	// --tail: under each agent, print the last non-empty transcript line — its
	// current activity. RAM/CPU alone can't tell a reasoning agent from a wedged
	// one; the live tail can (a thinking agent's last line keeps moving).
	tail := f.Bool("tail")
	textRuntime := map[string]bool{} // runtime name -> no usage_format (buffers to exit)
	// One task-tree scan for every live agent's blocked check, not one per
	// agent (store.BuildTaskIndex — the same discipline eventlog.Sync and
	// acceptance.go follow). A failed build degrades to nil: agentstate.Derive
	// then just never reports "blocked", never an error.
	tasks, _ := store.BuildTaskIndex(w)

	live, err := liveAgents(w)
	if err != nil {
		return err
	}
	liveRunIDs := map[string]bool{}
	for _, rec := range live {
		liveRunIDs[rec.RunID] = true
		u := procmon.SampleGroup(rec.PGID)
		age := time.Since(rec.Started).Round(time.Second)
		over := ""
		if maxRSS > 0 && int64(u.RSSKB)*1024 > maxRSS {
			over += fmt.Sprintf(" OVER-RAM(>%s)", humanBytes(maxRSS))
		}
		if maxRun > 0 && age > maxRun {
			over += fmt.Sprintf(" OVER-TIME(>%s)", maxRun)
		}
		// state is the same thinking/acting/waiting/stalled/blocked/silent label
		// the dashboard shows (agentstate.Derive is the single shared source) —
		// RAM/CPU/uptime alone can't tell a reasoning agent from a wedged one.
		// Printed uppercase for the states that want an operator's attention, so
		// a stalled agent stands out from a busy one without needing --tail.
		state := agentstate.Derive(w, rec, tasks)
		// A child that raised the break-glass BLOCKED channel is tagged distinctly
		// and its reason printed below — a run reporting it cannot run dacli must
		// never read as a normal live agent (task 269). The tag is kept OUT of
		// `over` so --reap (for RAM/time runaways) never kills an agent whose only
		// signal is that it asked for help. It composes with agentstate: Derive
		// reads the task's outstanding ask, this reads the run's blocked.txt.
		blocked := readBlocked(w, rec.RunID)
		status := over
		if _, reason := runLifecycleLive(w, rec, lifecycleNow()); reason != "process live" {
			status += " " + strings.ToUpper(strings.ReplaceAll(reason, " ", "-"))
		}
		if blocked != "" {
			status += " BLOCKED"
		}
		handoffRequired := store.RootHandoffRequested(w, rec.RunID)
		if _, err := os.Stat(store.RootHandoffPathForRun(w, rec.RunID)); err == nil {
			handoffRequired = true
		}
		if handoffRequired {
			status += " HANDOFF-REQUIRED"
		}
		// CPUPct is ps's %cpu: cputime/elapsed AVERAGED over each process's whole
		// lifetime, NOT an instantaneous sample. Labelled "CPUavg" so an operator
		// does not read a long-idle agent's high lifetime average as current load.
		fmt.Fprintf(ctx.Stdout, "%s  %-14s %-12s %-10s pid %-7d %2d proc  %8s RAM  %5.0f%% CPUavg  %7s GPU  up %s  [%s]%s\n",
			rec.RunID[:min(10, len(rec.RunID))], clikit.OrDash(rec.Child), clikit.OrDash(rec.Runtime),
			"task "+clikit.OrDash(rec.Task), rec.PID, u.Procs, humanKB(u.RSSKB), u.CPUPct, gpuStr(u.GPUMiB), age, stateLabel(state), status)
		if blocked != "" {
			fmt.Fprintf(ctx.Stdout, "            ⚠ BLOCKED: %s\n", truncateLine(firstLine(blocked), 100))
		}
		if handoffRequired {
			fmt.Fprintln(ctx.Stdout, "            ⚠ HANDOFF-REQUIRED: root must re-observe and consume the structured handoff")
		}
		if tail {
			line := tailLine(w, filepath.Join(w.RunDir(rec.RunID), "transcript.log"), rec.Runtime, textRuntime)
			fmt.Fprintf(ctx.Stdout, "            ↳ %s\n", truncateLine(line, 100))
		}
		if over != "" && reap {
			killOne(ctx, w, rec, 3*time.Second)
		}
	}
	if len(live) == 0 {
		fmt.Fprintln(ctx.Stdout, "no live agents")
	}
	for _, handoff := range pendingRootHandoffs(w) {
		if liveRunIDs[handoff.RunID] {
			continue
		}
		fmt.Fprintf(ctx.Stdout, "%s  %-14s task %-10s  HANDOFF-REQUIRED\n            ↳ %s\n",
			clikit.Short(handoff.RunID, 10), clikit.OrDash(handoff.ChildID), clikit.OrDash(handoff.TaskID), truncateLine(handoff.NextAction, 100))
	}
	return nil
}

type agentProgressView struct {
	Schema     string                `json:"schema"`
	Version    int                   `json:"version"`
	ObservedAt time.Time             `json:"observed_at"`
	Workers    []store.WorkerExplain `json:"workers"`
}

// renderAgentProgress consumes the same store projection as `explain`; it does
// not infer a parallel worker lifecycle from PID/RAM presentation details.
func renderAgentProgress(ctx *clikit.Ctx, w *workspace.Workspace, project string, now time.Time) error {
	projects := []string{project}
	if project == "" {
		projects = nil
		all, err := store.ListProjects(w)
		if err != nil {
			return err
		}
		for _, item := range all {
			projects = append(projects, item.Slug)
		}
	}
	view := agentProgressView{Schema: store.ProgressExplainSchema, Version: 1, ObservedAt: now.UTC(), Workers: []store.WorkerExplain{}}
	for _, slug := range projects {
		projection, err := store.ExplainProject(w, slug, now)
		if err != nil {
			return err
		}
		view.Workers = append(view.Workers, projection.Workers...)
	}
	sort.Slice(view.Workers, func(i, j int) bool { return view.Workers[i].RunID.Value < view.Workers[j].RunID.Value })
	if ctx.JSON {
		return clikit.EmitJSON(ctx, view)
	}
	if len(view.Workers) == 0 {
		fmt.Fprintln(ctx.Stdout, "no recorded workers for project")
		return nil
	}
	for _, worker := range view.Workers {
		fmt.Fprintf(ctx.Stdout, "%s agent=%s task=%s role=%s runtime=%s state=%s (source=%s observed=%s stale=%t)\n  claims: %s\n  next: %s\n",
			clikit.Short(worker.RunID.Value, 10), clikit.OrDash(worker.AgentID.Value), clikit.OrDash(worker.TaskID.Value), clikit.OrDash(worker.Role.Value), clikit.OrDash(worker.Runtime.Value), worker.State.Value,
			worker.State.Source, worker.State.ObservedAt.Format(time.RFC3339), worker.State.Stale, clikit.OrDash(strings.Join(worker.Claims.Value, ", ")), worker.NextAction.Value)
	}
	return nil
}

// stateLabel renders an agentstate.Derive result for the agents list: the
// three "needs a look" states (stalled/silent/blocked) are uppercased so they
// read as distinct from the three healthy ones (thinking/acting/waiting) at a
// glance — the same all-caps-for-attention convention OVER-RAM/OVER-TIME
// already use on this line.
func stateLabel(state string) string {
	switch state {
	case agentstate.Stalled, agentstate.Silent, agentstate.Blocked:
		return strings.ToUpper(state)
	default:
		return state
	}
}

// tailLine resolves what `agents --tail` shows under one agent: the
// transcript's last rendered line, or — when there is none yet — a note that
// tells a text runtime (whose child fully-buffers stdout until it exits) apart
// from a stream-json runtime that simply has nothing new to show.
func tailLine(w *workspace.Workspace, transcriptPath, runtimeName string, cache map[string]bool) string {
	if line := lastTranscriptLine(transcriptPath); line != "" {
		return line
	}
	if isTextRuntime(w, runtimeName, cache) {
		return "(text runtime — output appears at exit)"
	}
	return "(no transcript output yet)"
}

// isTextRuntime reports whether runtime name has no usage_format set — a text
// runtime whose child CLI fully-buffers stdout, so transcript.log stays empty
// until the process exits (not "stuck"). cache memoizes the LoadRuntime lookup
// across the agents list. An unresolvable name (empty, or no such adapter)
// reports false so --tail falls back to the generic no-output message.
func isTextRuntime(w *workspace.Workspace, name string, cache map[string]bool) bool {
	if name == "" {
		return false
	}
	if v, ok := cache[name]; ok {
		return v
	}
	rt, err := store.LoadRuntime(w, name)
	textOnly := err == nil && rt.UsageFormat == ""
	cache[name] = textOnly
	return textOnly
}

// parseBytes reads a size like "2G", "500M", "1024K", or a bare byte count.
func parseBytes(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	mult := int64(1)
	switch last := s[len(s)-1]; last {
	case 'G', 'g':
		mult, s = 1<<30, s[:len(s)-1]
	case 'M', 'm':
		mult, s = 1<<20, s[:len(s)-1]
	case 'K', 'k':
		mult, s = 1<<10, s[:len(s)-1]
	}
	n, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return int64(n * float64(mult))
}

func humanBytes(b int64) string { return humanKB(int(b / 1024)) }

// parseDurationArg reads "15m"/"2h"/"90s" or a bare seconds count.
func parseDurationArg(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Second
	}
	return 0
}

// cmdLogs prints, or with -f follows, a run's transcript. A detached child
// streams straight to the transcript file, so -f tails a live agent's output
// the way `tail -f` would — the missing "what is it actually doing" view.
func cmdLogs(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, err := clikit.ParseFlags(args, "tail")
	if err != nil {
		return err
	}
	if err := f.Reject("f", "follow", "tail"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli logs <run-id-prefix|child-id> [-f] [--tail N]")
	}
	ref := f.Pos[0]
	rec, haveRec := readProcByRef(w, ref)
	runID := rec.RunID
	if !haveRec {
		entries, _ := os.ReadDir(w.RunsDir())
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), ref) {
				runID = e.Name()
				break
			}
		}
	}
	if runID == "" {
		return store.ErrNotFound{Ref: "run " + ref}
	}
	path := filepath.Join(w.RunDir(runID), "transcript.log")

	data, _ := os.ReadFile(path)
	if n, err := f.Int("tail", 0); err != nil {
		return err
	} else if n <= 0 && len(f.All("tail")) > 0 {
		return clikit.Usagef("--tail must be a positive integer, got %d", n)
	} else if n > 0 {
		data = lastLines(data, n)
	}
	// Detached stream-json runs write RAW JSON events to the transcript (the tee
	// only runs on the foreground path), so render each line to readable text on
	// read — logs and -f show the same legible output as a text runtime.
	renderTranscriptTo(ctx.Stdout, data)
	var offset int64
	if fi, e := os.Stat(path); e == nil {
		offset = fi.Size()
	}
	if !(f.Bool("f") || f.Bool("follow")) {
		return nil
	}
	// Follow: drain appended bytes until the agent's process is gone (one final
	// drain after it exits), so the tail ends when the work does. Advance the
	// offset only to the last newline so a JSON event line is never split across
	// two renders.
	drain := func(final bool) {
		fi, e := os.Stat(path)
		if e != nil || fi.Size() <= offset {
			return
		}
		chunk := make([]byte, fi.Size()-offset)
		fh, e2 := os.Open(path)
		if e2 != nil {
			return
		}
		n, _ := fh.ReadAt(chunk, offset)
		_ = fh.Close()
		chunk = chunk[:n]
		if !final {
			nl := bytes.LastIndexByte(chunk, '\n')
			if nl < 0 {
				return // no complete line yet; wait for the rest
			}
			chunk = chunk[:nl+1]
		}
		renderTranscriptTo(ctx.Stdout, chunk)
		offset += int64(len(chunk))
	}
	for {
		time.Sleep(700 * time.Millisecond)
		drain(false)
		if !(haveRec && procmon.AliveRecord(rec)) {
			drain(true) // flush any trailing partial line once the work is done
			return nil
		}
	}
}

// renderTranscriptTo writes b to out with each complete line rendered from
// stream-json to readable text (assistant text / [tool: X] markers); a
// plain-text line passes through unchanged and blank lines are dropped. This is
// the read-side counterpart of teeStreamJSON: it makes a detached run's raw
// stream-json transcript as legible as a foreground run's already-teed one.
func renderTranscriptTo(out io.Writer, b []byte) {
	for _, ln := range bytes.Split(b, []byte("\n")) {
		if len(bytes.TrimSpace(ln)) == 0 {
			continue
		}
		text, _ := renderStreamLine(ln)
		if text == "" {
			text, _ = renderCodexLine(ln, streamUsage{})
		}
		if text != "" {
			fmt.Fprintln(out, text)
		}
	}
}

// lastTranscriptLine reads path and returns its most recent readable line — the
// agent's current activity for `dacli agents --tail`. A detached stream-json
// child writes raw JSON events here, so each candidate line is rendered on read
// (assistant text / [tool: X]); events with no human-facing content are skipped.
// Missing/empty file yields "".
func lastTranscriptLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	// walk backwards for the last line that renders to non-empty text.
	end := len(data)
	for end > 0 {
		start := bytes.LastIndexByte(data[:end], '\n')
		raw := bytes.TrimSpace(data[start+1 : end])
		if len(raw) > 0 {
			text, _ := renderStreamLine(raw)
			if text == "" {
				text, _ = renderCodexLine(raw, streamUsage{})
			}
			if text != "" {
				// A rendered assistant event may span lines; the current activity
				// is its last line.
				if i := strings.LastIndexByte(text, '\n'); i >= 0 {
					text = text[i+1:]
				}
				return text
			}
		}
		if start < 0 {
			break
		}
		end = start
	}
	return ""
}

// truncateLine shortens s to at most max runes, appending an ellipsis when cut.
func truncateLine(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// lastLines returns the last n newline-delimited lines of b.
func lastLines(b []byte, n int) []byte {
	count := 0
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == '\n' {
			count++
			if count > n {
				return b[i+1:]
			}
		}
	}
	return b
}

// splitClaims parses one comma-separated --claim value into cleaned paths.
func splitClaims(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// splitClaimValues accumulates every repeated --claim occurrence while also
// supporting comma-separated paths within each occurrence. Order is preserved
// so preview output matches the operator's invocation and the durable record.
func splitClaimValues(values []string) []string {
	var out []string
	for _, value := range values {
		out = append(out, splitClaims(value)...)
	}
	return out
}

// cmdKill terminates one agent's whole process tree, or --all of them. The
// group is SIGTERM'd, then SIGKILL'd after a grace window if anything survives
// — so a well-behaved agent exits cleanly and a hung one is still guaranteed
// dead, with no orphaned children left holding resources.
func cmdKill(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	// Terminating process groups is a privileged, irreversible side effect (and
	// the target pgid comes from an on-disk record any rw child can forge) —
	// keep it off the read-only surface.
	if err := clikit.RequireRW(id, "killing an agent"); err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("all", "grace", "run", "child"); err != nil {
		return err
	}
	grace := time.Duration(3) * time.Second
	if n, err := f.Int("grace", 0); err != nil {
		return err
	} else if n > 0 {
		grace = time.Duration(n) * time.Second
	}

	if f.Bool("all") {
		live, err := liveAgents(w)
		if err != nil {
			return err
		}
		if len(live) == 0 {
			fmt.Fprintln(ctx.Stdout, "no live agents to kill")
			return nil
		}
		for _, rec := range live {
			killOne(ctx, w, rec, grace)
		}
		return nil
	}

	ref := ""
	if len(f.Pos) > 0 {
		ref = f.Pos[0]
	} else if r := f.Get("run"); r != "" {
		ref = r
	} else if c := f.Get("child"); c != "" {
		ref = c
	}
	if ref == "" {
		return clikit.Usagef("usage: dacli kill <run-id-prefix | child-id> [--grace sec]  |  dacli kill --all")
	}
	live, err := liveAgents(w)
	if err != nil {
		return err
	}
	for _, rec := range live {
		if strings.HasPrefix(rec.RunID, ref) || rec.Child == ref {
			killOne(ctx, w, rec, grace)
			return nil
		}
	}
	return store.ErrNotFound{Ref: "live agent " + ref}
}

// pidStart captures a freshly-started child's OS start time for its proc.txt,
// so a later reader can tell the real agent from a process that recycled its
// PID. Best-effort: an empty string just falls back to a bare liveness probe.
func pidStart(pid int) string { s, _ := procmon.ProcStart(pid); return s }

// liveAgents reads every run's proc.txt and returns those whose leader process
// is still alive AND still identifies as the spawned agent (PID not recycled),
// newest first.
//
// "No runs yet" (the directory does not exist) is normal and returns an empty
// result with no error. A directory that exists but cannot be read is a
// different fact — reporting both as "no live agents" hid the second entirely
// and read as "nobody is working" to every caller, including the WIP-facing
// callers (`agents`, `kill --all`, `wait`) that decide from this result
// whether anyone is currently active (dacli 337; mirrors cmdRunsList, which
// could already tell the two apart).
func liveAgents(w *workspace.Workspace) ([]procmon.Record, error) {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot read the runs directory at %s: %w", w.RunsDir(), err)
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names))) // ULIDs: newest first
	var out []procmon.Record
	for _, n := range names {
		procPath := filepath.Join(w.RunDir(n), "proc.txt")
		rec, err := procmon.ReadRecord(procPath)
		if err != nil {
			continue
		}
		if live, _ := runLifecycleLive(w, rec, lifecycleNow()); live {
			out = append(out, rec)
		} else if rec.Outcome == "" && len(rec.Claims) > 0 {
			// Crash recovery may release a stale claim only after the shared
			// lifecycle classifier has rejected both the recorded process
			// identity and the bounded activity grace. The terminal transition
			// is atomic, so a concurrent claim check never sees half a release.
			if err := procmon.CompleteRecord(procPath, rec, recoveredExitOutcome); err != nil {
				return nil, fmt.Errorf("record critical run artifact proc.txt: %w", err)
			}
		}
	}
	return out, nil
}

func killOne(ctx *clikit.Ctx, w *workspace.Workspace, rec procmon.Record, grace time.Duration) {
	// Reconcile again immediately before the irreversible action. Callers
	// normally supply liveAgents output, but a stale record must never make a
	// recycled numeric process group a signal target (task 369).
	if !procmon.ReconcileRun(rec) {
		fmt.Fprintf(ctx.Stdout, "%s: nothing to signal (already gone)\n", clikit.OrDash(rec.Child))
		return
	}
	before := procmon.SampleGroup(rec.PGID).Procs
	if !terminateRecordedTree(rec, grace) {
		fmt.Fprintf(ctx.Stdout, "%s: nothing to signal (already gone)\n", clikit.OrDash(rec.Child))
		return
	}
	verb := "SIGTERM→SIGKILL if needed"
	// Audit crumb next to the run record: what was reaped and how.
	openRunRecord(w.RunDir(rec.RunID), ctx.Stderr).bestEffort("killed.txt",
		fmt.Sprintf("killed %s (pgid %d, ~%d proc) via %s at %s\n",
			rec.Child, rec.PGID, before, verb, time.Now().UTC().Format(time.RFC3339)))
	fmt.Fprintf(ctx.Stdout, "killed %s — process group %d (~%d proc) reaped via %s\n",
		clikit.OrDash(rec.Child), rec.PGID, before, verb)
}

// cmdWait blocks until the named detached run(s) finish — or all live agents if
// none are named — then finalizes each one's outcome from the workspace effects
// it left behind. This is the block half of async orchestration: `spawn
// --detach` many, then `wait` on them, instead of hand-rolling shell polling.
// runStillLive reports whether the recorded leader still identifies the live
// process tree. Descendants remain covered while that leader is alive. After
// it exits, however, the bare numeric PGID can be reused and can no longer be
// safely attributed or signalled; task 369 closes that residual from task 285.
func runStillLive(rec procmon.Record) bool {
	return procmon.ReconcileRun(rec)
}

const (
	runStartupGrace       = 30 * time.Second
	transcriptActiveGrace = 15 * time.Second
	recoveredExitOutcome  = "process exited (recovered)"
)

// runLifecycleLive is the shared startup/completion view for agents and wait.
// The recorded guardian is normally authoritative, but Codex runs 01KZVSF64J
// and 01KZVSFBXR showed that its CLI can disappear during registration while a
// worker continues writing the inherited transcript descriptor. A short,
// explicit startup grace prevents that ordering window from becoming a false
// completion; recent transcript writes are durable cross-process evidence that
// work continues after it. Both bounds are deliberately finite, so a launch
// with neither a process nor advancing output still becomes finalizable.
func runLifecycleLive(w *workspace.Workspace, rec procmon.Record, now time.Time) (bool, string) {
	// proc.txt is the terminal lifecycle authority shared by recovery and wait.
	// agents can stamp this outcome while outcome.md still says running and a
	// detached writer still advances its transcript; neither secondary signal
	// may resurrect the completed record (task 436).
	if rec.Outcome != "" {
		return false, ""
	}
	if raw, err := os.ReadFile(filepath.Join(w.RunDir(rec.RunID), "outcome.md")); err == nil {
		first, _, _ := strings.Cut(string(raw), "\n")
		if strings.HasPrefix(first, "outcome:") && first != detachedRunningPlaceholder {
			return false, ""
		}
	}
	// A durable watchdog verdict outranks every inferred liveness signal. In
	// particular, the timeout marker can be written while the run is still
	// young enough for startup grace; retaining it here leaks the task's path
	// claim even though the watchdog already killed and finalized the tree.
	if _, err := os.Stat(filepath.Join(w.RunDir(rec.RunID), timeoutMarker)); err == nil {
		return false, ""
	}
	// A governed kill records this marker only after the process tree has been
	// reaped. Reconcile the recorded identity once more before trusting it: the
	// marker is durable intent, while the process check prevents a partial or
	// forged marker from declaring a still-running worker terminal.
	if _, err := os.Stat(filepath.Join(w.RunDir(rec.RunID), "killed.txt")); err == nil && !runStillLive(rec) {
		return false, ""
	}
	// The guardian writes this only after Wait returns, so it is stronger
	// termination evidence than a process-table miss. Check it before transcript
	// activity: the final write commonly lands immediately before this marker.
	if _, err := os.Stat(filepath.Join(w.RunDir(rec.RunID), "runtime-exit.txt")); err == nil {
		return false, ""
	}
	if runStillLive(rec) {
		return true, "process live"
	}
	// For an unobservable identity the configured deadline is the finite upper
	// bound on transcript-derived liveness. The watchdog normally leaves a
	// marker, but recovery must remain correct if it could not do so.
	if rec.Timeout > 0 && !rec.Started.IsZero() && !now.Before(rec.Started.Add(rec.Timeout)) {
		return false, ""
	}
	if age := now.Sub(rec.Started); !rec.Started.IsZero() && age >= 0 && age < runStartupGrace {
		return true, "startup grace"
	}
	if info, err := os.Stat(filepath.Join(w.RunDir(rec.RunID), "transcript.log")); err == nil && info.Size() > 0 {
		if age := now.Sub(info.ModTime()); age >= 0 && age < transcriptActiveGrace {
			return true, "transcript active"
		}
		// Issue #672 showed transcript gaps longer than the fixed freshness
		// window while Codex was still editing and testing. Once output proves a
		// worker started, retain that evidence through the configured runtime
		// timeout unless the guardian records its real exit above. Legacy records
		// without a timeout deliberately retain the bounded freshness behavior.
		if rec.Timeout > 0 && !rec.Started.IsZero() && now.Before(rec.Started.Add(rec.Timeout)) {
			return true, "transcript active"
		}
	}
	return false, ""
}

func cmdWait(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("interval", "timeout"); err != nil {
		return err
	}
	result := commandresult.Wait{}
	ctx.Result = result
	interval := 3 * time.Second
	if n, err := f.Int("interval", 0); err != nil {
		return err
	} else if n > 0 {
		interval = time.Duration(n) * time.Second
	}
	overall := 3600
	if n, err := f.Int("timeout", 0); err != nil {
		return err
	} else if n > 0 {
		overall = n
	}

	pending := map[string]procmon.Record{}
	if len(f.Pos) > 0 {
		for _, ref := range f.Pos {
			if rec, ok := readProcByRef(w, ref); ok {
				// A recovered process miss still needs effects-based finalization.
				// Every other proc outcome is already terminal; waiting on it again
				// must not duplicate retirement, outcome writes, or exit events.
				if rec.Outcome == "" || rec.Outcome == recoveredExitOutcome {
					pending[rec.RunID] = rec
				}
			} else {
				fmt.Fprintf(ctx.Stderr, "no run matching %q\n", ref)
			}
		}
	} else {
		live, err := liveAgents(w)
		if err != nil {
			return err
		}
		for _, rec := range live {
			pending[rec.RunID] = rec
		}
	}
	if len(pending) == 0 {
		fmt.Fprintln(ctx.Stdout, "nothing to wait for")
		return nil
	}

	// Startup line: name how many runs we are waiting on and their short ids, so a
	// foreground wait shows what it is blocking on the moment it begins.
	total := len(pending)
	ids := make([]string, 0, total)
	for id := range pending {
		ids = append(ids, id[:min(10, len(id))])
	}
	sort.Strings(ids)
	fmt.Fprintf(ctx.Stdout, "waiting on %d run(s): %s\n", total, strings.Join(ids, ", "))

	// Light heartbeat: between completions the loop is silent for the whole
	// interval gap, so a long wait looks dead. Every ~30s (not every poll) print
	// one line proving the wait is still alive, without spamming.
	start := time.Now()
	nextBeat := start.Add(30 * time.Second)
	deadline := start.Add(time.Duration(overall) * time.Second)
	var runFailures []error
	for len(pending) > 0 {
		pendingIDs := make([]string, 0, len(pending))
		for id := range pending {
			pendingIDs = append(pendingIDs, id)
		}
		sort.Strings(pendingIDs)
		for _, id := range pendingIDs {
			rec := pending[id]
			// A run that raised the BLOCKED channel is finalized immediately, even
			// while its process is still live: it has told us it is stuck and will
			// not self-complete, so waiting on it as if it might is precisely the
			// silence task 269 removes. finalizeRun reports it as BLOCKED.
			if live, _ := runLifecycleLive(w, rec, lifecycleNow()); !live || readBlocked(w, id) != "" {
				summary, finalizeErr := finalizeRunChecked(w, rec)
				if finalizeErr != nil {
					return finalizeErr
				}
				fmt.Fprintf(ctx.Stdout, "%s  %s (%d of %d)\n", id[:min(10, len(id))], summary, total-len(pending)+1, total)
				outcome := "finalized"
				if completed, readErr := procmon.ReadRecord(filepath.Join(w.RunDir(id), "proc.txt")); readErr == nil && completed.Outcome != "" {
					outcome = completed.Outcome
				}
				result.Runs = append(result.Runs, commandresult.WaitRun{RunID: id, Child: rec.Child, Outcome: outcome})
				ctx.Result = result
				if failure := detachedRuntimeFailure(w, rec); failure != nil {
					runFailures = append(runFailures, fmt.Errorf("detached run %s: %w", id[:min(10, len(id))], failure))
				}
				delete(pending, id)
			}
		}
		if len(pending) == 0 {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("wait timed out with %d run(s) still live (raise --timeout, or dacli kill them)", len(pending))
		}
		if now := time.Now(); now.After(nextBeat) {
			fmt.Fprintf(ctx.Stdout, "still waiting on %d run(s) (up %s)\n", len(pending), now.Sub(start).Round(time.Second))
			nextBeat = now.Add(30 * time.Second)
		}
		time.Sleep(interval)
	}
	return errors.Join(runFailures...)
}

// detachedRuntimeFailure reconstructs the governed provider error after a
// detached guardian has exited. The guardian's numeric marker and bounded tail
// are durable; an exec.ExitError is not, so commandresult supplies a recorded
// exit cause with the same stable diagnostic fields (issue #876).
func detachedRuntimeFailure(w *workspace.Workspace, rec procmon.Record) error {
	rawExit, readErr := os.ReadFile(filepath.Join(w.RunDir(rec.RunID), "runtime-exit.txt"))
	if readErr != nil {
		if errors.Is(readErr, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read detached runtime exit marker: %w", readErr)
	}
	var exitCode int
	if _, scanErr := fmt.Sscanf(strings.TrimSpace(string(rawExit)), "%d", &exitCode); scanErr != nil {
		return fmt.Errorf("parse detached runtime exit marker: %w", scanErr)
	}
	if exitCode == 0 {
		return nil
	}
	binary := rec.Runtime
	if rt, err := store.LoadRuntime(w, rec.Runtime); err == nil && rt.Binary != "" {
		binary = rt.Binary
	}
	if binary == "" {
		binary = "runtime"
	}
	cmd := exec.Command(binary)
	cmd.Dir = w.Root
	tail, tailErr := readRuntimeDiagnosticTail(filepath.Join(w.RunDir(rec.RunID), "transcript.log"))
	externalErr := commandresult.NewRecordedExitError(cmd, commandresult.RunOptions{
		Operation: "runtime " + rec.Runtime + " detached launch", WorkspaceRoot: w.Root,
	}, nil, tail, exitCode)
	if tailErr != nil {
		return fmt.Errorf("read detached runtime transcript tail: %w", errors.Join(tailErr, externalErr))
	}
	return externalErr
}

func readRuntimeDiagnosticTail(path string) ([]byte, error) {
	const limit = 8 << 10
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > limit {
		if _, seekErr := f.Seek(info.Size()-limit, io.SeekStart); seekErr != nil {
			return nil, seekErr
		}
	}
	return io.ReadAll(io.LimitReader(f, limit))
}

// readProcByRef finds any run (live or finished) whose id-prefix or child id
// matches ref.
func readProcByRef(w *workspace.Workspace, ref string) (procmon.Record, bool) {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return procmon.Record{}, false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(e.Name()), "proc.txt"))
		if err != nil {
			continue
		}
		if strings.HasPrefix(rec.RunID, ref) || rec.Child == ref {
			return rec, true
		}
	}
	return procmon.Record{}, false
}

// finalizeRun computes a finished detached run's outcome from what it actually
// wrote to the workspace (acceptance boxes + events by the child), overwriting
// the "running (detached)" placeholder. A detached child is not our OS child,
// so there is no exit code to read — the outcome is derived from effects, which
// is the honest thing to report.
func finalizeRun(w *workspace.Workspace, rec procmon.Record) string {
	summary, err := finalizeRunChecked(w, rec)
	if err != nil {
		return fmt.Sprintf("%s: finalization failed: %v", rec.Child, err)
	}
	return summary
}

func finalizeRunChecked(w *workspace.Workspace, rec procmon.Record) (string, error) {
	runDir := w.RunDir(rec.RunID)
	record := openRunRecord(runDir, nil)
	workDir := w.Root
	isolatedWorktree := false
	if raw, e := os.ReadFile(filepath.Join(runDir, "worktree.txt")); e == nil {
		candidate := strings.TrimSpace(string(raw))
		if candidate != "" {
			workDir, isolatedWorktree = candidate, true
		}
	}
	plannedHandoff := false
	if raw, err := os.ReadFile(filepath.Join(runDir, "planned-handoffs.txt")); err == nil && strings.TrimSpace(string(raw)) != "" {
		plannedHandoff = true
	}
	// The independent watchdog owns the timed-out verdict. A concurrently
	// polling `wait` may observe the now-dead tree immediately afterwards; it
	// must not overwrite that durable verdict with effects-derived "done" or
	// "no visible result" (task 372).
	if _, err := os.Stat(filepath.Join(runDir, timeoutMarker)); err == nil {
		if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, "timed out"); err != nil {
			return "", fmt.Errorf("record critical run artifact proc.txt: %w", err)
		}
		if rec.Child != "" {
			_ = store.RetireAgent(w, rec.Child)
		}
		return fmt.Sprintf("%s: timed out after %s", rec.Child, rec.Timeout), nil
	}
	// Free the agent's WIP slot only after its terminal artifacts are durable.
	// Nothing in the
	// lifecycle called RetireAgent, so every spawn's agent held capacity
	// forever: roles filled to their limit and later spawns were refused while
	// `dacli agents` showed nobody live (task 282). ActiveInRole no longer
	// COUNTS a finished agent either — that half is self-healing for the
	// backlog already leaked — but retiring here keeps the roster honest about
	// which agents are done rather than merely inferred-done.
	//
	// The retirement itself remains best-effort: an agent file that cannot be
	// written must never invalidate an otherwise durable terminal run record.
	// The break-glass BLOCKED channel wins over any derived outcome: a child that
	// raised it told us, in its own words, that it could not run dacli. Reporting
	// that run as "done" or "no visible result" would bury exactly the failure the
	// channel exists to surface, so BLOCKED is stamped and returned first (269).
	if reason := readBlocked(w, rec.RunID); reason != "" {
		elapsed := time.Since(rec.Started).Round(time.Second)
		if isolatedWorktree || store.RootHandoffRequested(w, rec.RunID) {
			handoff, required, captureErr := store.CaptureRootHandoff(w, rec.RunID, rec.Task, rec.Child, workDir, store.RootHandoffRequest{
				Schema: store.RootHandoffSchema, FailedOperation: "worker lifecycle publication", FailureClass: "policy_refusal", Stderr: reason,
				NextAction: "owner consumes the handoff after hash re-observation, reruns verification, then commits and publishes without changing worker harness or grant",
			}, time.Now())
			if captureErr != nil {
				return "", fmt.Errorf("capture root handoff: %w", captureErr)
			}
			if required {
				if err := record.critical("outcome.md", fmt.Sprintf("outcome: handoff-required (detached)\nchild: %s\nelapsed_since_start: %s\nfailed_operation: %s\nnext: %s\n", rec.Child, elapsed, handoff.FailedOperation, handoff.NextAction)); err != nil {
					return "", err
				}
				if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, "handoff-required"); err != nil {
					return "", fmt.Errorf("record critical run artifact proc.txt: %w", err)
				}
				if rec.Child != "" {
					_ = store.RetireAgent(w, rec.Child)
				}
				recordExit(w, rec, "handoff-required", elapsed, handoff.FailedOperation)
				return fmt.Sprintf("%s: handoff-required — %s", rec.Child, handoff.NextAction), nil
			}
		}
		if err := record.critical("outcome.md", fmt.Sprintf("outcome: blocked (detached)\nchild: %s\nelapsed_since_start: %s\nreason: %s\n",
			rec.Child, elapsed, reason)); err != nil {
			return "", err
		}
		if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, "blocked"); err != nil {
			return "", fmt.Errorf("record critical run artifact proc.txt: %w", err)
		}
		if rec.Child != "" {
			_ = store.RetireAgent(w, rec.Child)
		}
		recordExit(w, rec, "blocked", elapsed, fmt.Sprintf("the child raised the BLOCKED channel: %s", firstLine(reason)))
		return fmt.Sprintf("%s: BLOCKED — %s", rec.Child, firstLine(reason)), nil
	}
	eventsWS := w
	if isolatedWorktree {
		raw := []byte(workDir)
		if wtw, e2 := workspace.Find(strings.TrimSpace(string(raw))); e2 == nil {
			eventsWS = wtw
		}
	}
	done, total := 0, 0
	if t, _ := store.FindTask(w, rec.Task); t != nil {
		for _, b := range t.Acceptance() {
			total++
			if b.Done {
				done++
			}
		}
	}
	childEvents, _ := eventlog.List(eventsWS, eventlog.Query{Actor: rec.Child})
	providerSummary := ""
	if _, err := os.Stat(filepath.Join(runDir, "provider-outcome.txt")); os.IsNotExist(err) {
		if raw, readErr := os.ReadFile(filepath.Join(runDir, "runtime-exit.txt")); readErr == nil {
			var exitCode int
			if _, scanErr := fmt.Sscanf(strings.TrimSpace(string(raw)), "%d", &exitCode); scanErr == nil && exitCode != 0 {
				var printed strings.Builder
				if policyErr := recordProviderOutcome(&printed, w, rec.Runtime, filepath.Join(runDir, "transcript.log"), exitCode, record.bestEffort); policyErr != nil {
					providerSummary = fmt.Sprintf("provider policy record failed: %v", policyErr)
				} else {
					providerSummary = strings.TrimSpace(printed.String())
				}
			}
		}
	}
	// A detached child streamed straight to transcript.log without an in-process
	// parser (the parent had already returned), so usage was never captured live.
	// If the transcript is a stream-json log, harvest its final usage now. Parsing
	// is self-detecting: a plain-text transcript yields no `result` event and
	// nothing is written, so text runtimes are unaffected.
	if _, err := os.Stat(filepath.Join(runDir, "usage.txt")); os.IsNotExist(err) {
		if f, e := os.Open(filepath.Join(runDir, "transcript.log")); e == nil {
			u := teeStreamJSON(f, io.Discard)
			if !u.found {
				_, _ = f.Seek(0, io.SeekStart)
				u = teeStructuredJSON(f, io.Discard, "codex-jsonl")
			}
			_ = f.Close()
			if u.found {
				writeUsage(runDir, u)
			}
		}
	}
	elapsed := time.Since(rec.Started).Round(time.Second)
	outcome := "done"
	var handoff store.RootHandoff
	handoffRequired := false
	if store.RootHandoffRequested(w, rec.RunID) || (isolatedWorktree && plannedHandoff) {
		var captureErr error
		handoff, handoffRequired, captureErr = store.CaptureRootHandoff(w, rec.RunID, rec.Task, rec.Child, workDir, store.RootHandoffRequest{
			Schema: store.RootHandoffSchema, FailedOperation: "worker lifecycle publication", FailureClass: "filesystem_sandbox_refusal",
			NextAction: "owner consumes the handoff after hash re-observation, reruns verification, then commits and publishes without changing worker harness or grant",
		}, time.Now())
		if captureErr != nil {
			return "", fmt.Errorf("capture root handoff: %w", captureErr)
		}
	}
	if handoffRequired {
		outcome = "handoff-required"
	} else if len(childEvents) == 0 && done == 0 {
		outcome = "no visible result"
	}
	if err := record.critical("outcome.md", fmt.Sprintf("outcome: %s (detached)\nchild: %s\nelapsed_since_start: %s\nacceptance: %d/%d\nevents_by_child: %d\n",
		outcome, rec.Child, elapsed, done, total, len(childEvents))); err != nil {
		return "", err
	}
	if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, outcome); err != nil {
		return "", fmt.Errorf("record critical run artifact proc.txt: %w", err)
	}
	if rec.Child != "" {
		_ = store.RetireAgent(w, rec.Child)
	}
	recordExit(w, rec, outcome, elapsed, fmt.Sprintf("wrote %d event(s), checked %d of %d acceptance box(es)",
		len(childEvents), done, total))
	summary := fmt.Sprintf("%s: %s · %s · %d event(s) · acceptance %d/%d",
		rec.Child, outcome, elapsed, len(childEvents), done, total)
	if providerSummary != "" {
		return providerSummary + " · " + summary, nil
	}
	if handoffRequired {
		summary += " · next: " + handoff.NextAction
	}
	return summary, nil
}

// recordExit writes the run's ending into the append-only log, so the fact that
// an agent finished — and with what result — survives independently of who
// later reads a run directory (issue #449).
//
// finalizeRun is reached exactly once per run: it is gated on an outcome that
// is missing or still the running placeholder, and it overwrites that outcome
// before returning. So the event is written once, not once per observation.
//
// Best-effort. A run whose ending cannot be logged is still finalized —
// refusing to finalize because the log write failed would restore precisely the
// invisible-run state this exists to end.
func recordExit(w *workspace.Workspace, rec procmon.Record, outcome string, elapsed time.Duration, detail string) {
	if rec.Child == "" {
		return // nothing to attribute the ending to
	}
	body := fmt.Sprintf("run %s ended: %s after %s — %s", rec.RunID, outcome, elapsed, detail)
	if rec.Role != "" {
		body += fmt.Sprintf(" (role %s)", rec.Role)
	}
	_, _ = eventlog.Append(w, rec.Child, model.EventExit, rec.Task, "run", body)
}

// detachedRunningPlaceholder is the exact first line of the outcome.md a
// `spawn --detach` writes before the run has finished. finalizeRun overwrites
// it; any run still holding it whose process is gone was never finalized —
// nobody ran `dacli wait` on it.
const detachedRunningPlaceholder = "outcome: running (detached)"

// humanKB renders a KB resident-set size as MiB/GiB.
func humanKB(kb int) string {
	mb := float64(kb) / 1024
	if mb >= 1024 {
		return fmt.Sprintf("%.1fGiB", mb/1024)
	}
	return fmt.Sprintf("%.0fMiB", mb)
}

// gpuStr renders GPU memory, honestly reporting n/a where it cannot be
// measured (no nvidia-smi) rather than a misleading 0.
func gpuStr(mib int) string {
	if mib < 0 {
		return "n/a"
	}
	return fmt.Sprintf("%dMiB", mib)
}

// warnMainCheckoutSpawn reports the two ways a no-worktree spawn leaves the
// repository unable to land work: no trunk to integrate into, or a main
// checkout already parked on a task branch.
//
// It warns rather than refuses. A no-worktree spawn is legitimate — a
// single-agent run, a read-only reviewer, a repo with no git at all — so
// refusing would break working setups to prevent a mistake the operator may
// not be making. What is not acceptable is silence, which is what produced a
// trunkless repo and an "integrated 0" that read as success.
func warnMainCheckoutSpawn(ctx *clikit.Ctx, w *workspace.Workspace) {
	if !gitx.Available() {
		return
	}
	if !hasAnyTrunk(w.Root) {
		fmt.Fprintf(ctx.Stderr, "warning: this repo has no trunk branch (main/master), and without --worktree the child branches in the MAIN checkout — `dacli ship` will then have nowhere to integrate into. Create a trunk first (git checkout -b main && git commit), or spawn with --worktree.\n")
		return
	}
	head, err := gitx.Run(w.Root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return
	}
	if h := strings.TrimSpace(head); strings.HasPrefix(h, "dacli/") {
		fmt.Fprintf(ctx.Stderr, "warning: the main checkout is on task branch %s, not trunk — work spawned here lands on that branch and `dacli ship` integrates from trunk. Switch back (git checkout main) or use --worktree so each agent gets its own branch.\n", h)
	}
}

func taskBranch(t *store.Task) string {
	return fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug)
}

// validateReviewTarget keeps local reviews local: unlike PR-first reviews,
// their target is the task's canonical branch and must exist before a child is
// minted. A missing branch used to surface only as a failed gh lookup inside
// the child, leaving a detached run with no visible result (issue #714).
func validateReviewTarget(w *workspace.Workspace, t *store.Task, f *clikit.Flags) error {
	if !f.Bool("review") || f.Bool("pr") {
		return nil
	}
	branch := taskBranch(t)
	if !gitx.Available() {
		return clikit.Refusedf("local review target %s needs git to inspect task branch %s — install git or re-run with --pr", t.ID, branch)
	}
	if !gitx.BranchExists(w.Root, branch) {
		return clikit.Refusedf("local review target %s has no task branch %s — create or restore it, or re-run with --pr", t.ID, branch)
	}
	return nil
}

// resolveSpawnWorkDir preserves a task checkout when spawn is invoked from
// inside it without --worktree. workspace.Find intentionally redirects that
// caller to the shared workspace root, so using w.Root as the runtime cwd loses
// the code checkout that owns the task branch (issue #673).
//
// Only Git's registered worktree/branch pair is trusted. Directory names are
// human-readable conveniences and can be renamed or copied; treating one as
// attribution would let a same-looking but unrelated tree receive the run.
func resolveSpawnWorkDir(w *workspace.Workspace, t *store.Task, cwd string, explicit bool) (string, bool, error) {
	if explicit {
		return w.WorktreePath(t.Project, t.Seq, t.Slug), true, nil
	}
	caller, root, wtRoot := resolvedPath(cwd), resolvedPath(w.Root), resolvedPath(w.WorktreesDir())
	if (caller == root || strings.HasPrefix(caller, root+string(filepath.Separator))) &&
		caller != wtRoot && !strings.HasPrefix(caller, wtRoot+string(filepath.Separator)) {
		// Preserve the historical no-git path for the main checkout. A plain
		// workspace (including unit-test fixtures) has no worktree registry to
		// consult, and an unrelated task worktree elsewhere must not pull a main
		// checkout spawn away from its caller.
		return w.Root, false, nil
	}
	wts, err := gitx.ListWorktrees(w.Root)
	if err != nil {
		return "", false, fmt.Errorf("cannot resolve spawn worktree: %w", err)
	}
	return resolveSpawnWorkDirFrom(w, t, cwd, false, wts)
}

// resolveSuperviseWorkDir preserves the main-checkout default except when root
// has durably reclaimed this task's checkout. In that recovery state the
// transfer identifies the canonical task worktree that correction turns must
// resume in; launching from main would lose governed commit attribution.
func resolveSuperviseWorkDir(w *workspace.Workspace, t *store.Task, cwd string) (string, bool, error) {
	workDir, isolated, err := resolveSpawnWorkDir(w, t, cwd, false)
	if err != nil || isolated || resolvedPath(cwd) != resolvedPath(w.Root) {
		return workDir, isolated, err
	}

	entries, err := os.ReadDir(w.RunsDir())
	if errors.Is(err, fs.ErrNotExist) {
		return workDir, isolated, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read recovery runs: %w", err)
	}
	var transferred string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		runDir := w.RunDir(entry.Name())
		rec, err := procmon.ReadRecord(filepath.Join(runDir, "proc.txt"))
		if err != nil || rec.Task != t.ID {
			continue
		}
		path, found, err := rootTransferredWorktree(filepath.Join(runDir, "worktree-transfer.txt"), taskBranch(t))
		if err != nil {
			return "", false, fmt.Errorf("read task recovery transfer in run %s: %w", rec.RunID, err)
		}
		if !found {
			continue
		}
		if transferred != "" && resolvedPath(transferred) != resolvedPath(path) {
			return "", false, clikit.Refusedf("task %s has multiple root-reclaimed worktrees; inspect recovery transfers before supervising", t.ID)
		}
		transferred = path
	}
	if transferred == "" {
		return workDir, isolated, nil
	}

	wts, err := gitx.ListWorktrees(w.Root)
	if err != nil {
		return "", false, fmt.Errorf("cannot resolve recovered task worktree: %w", err)
	}
	wantBranch := taskBranch(t)
	var candidates []gitx.Worktree
	for _, wt := range wts {
		if wt.Branch == wantBranch {
			candidates = append(candidates, wt)
		}
	}
	if len(candidates) != 1 || resolvedPath(candidates[0].Path) != resolvedPath(transferred) {
		// Reuse the established ambiguity refusal when it applies; otherwise a
		// transfer that no longer names the registered canonical branch is not
		// sufficient authority to redirect a root-owned correction.
		if len(candidates) > 1 {
			return resolveSpawnWorkDirFrom(w, t, transferred, false, wts)
		}
		return "", false, clikit.Refusedf("root recovery transfer for task %s does not name its registered canonical branch %s", t.ID, wantBranch)
	}
	return filepath.Clean(candidates[0].Path), true, nil
}

// rootTransferredWorktree reads only the binding needed to resume a root-owned
// recovery. The vcs slice owns writing this small text record; execution keeps
// its reader local to preserve feature-slice isolation.
func rootTransferredWorktree(path, wantBranch string) (string, bool, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	if values["new_owner"] != agentid.RootID {
		return "", false, nil
	}
	if values["worktree"] == "" || values["branch"] == "" {
		return "", false, fmt.Errorf("missing worktree or branch")
	}
	if values["branch"] != wantBranch {
		return "", false, clikit.Refusedf("recovery transfer branch %s does not match task branch %s", values["branch"], wantBranch)
	}
	return filepath.Clean(values["worktree"]), true, nil
}

func resolveSpawnWorkDirFrom(w *workspace.Workspace, t *store.Task, cwd string, explicit bool, wts []gitx.Worktree) (string, bool, error) {
	if explicit {
		return w.WorktreePath(t.Project, t.Seq, t.Slug), true, nil
	}
	wantBranch := taskBranch(t)
	var candidates []gitx.Worktree
	for _, wt := range wts {
		if wt.Branch == wantBranch {
			candidates = append(candidates, wt)
		}
	}
	if len(candidates) > 1 {
		return "", false, clikit.Refusedf("task %s has multiple registered worktrees on %s; refusing to choose between them — inspect `git worktree list`, then run `git worktree remove <duplicate-path>` and retry", t.ID, wantBranch)
	}

	caller := resolvedPath(cwd)
	var current *gitx.Worktree
	for i := range wts {
		path := resolvedPath(wts[i].Path)
		if caller == path || strings.HasPrefix(caller, path+string(filepath.Separator)) {
			// Prefer the most specific registration if a repository is nested in
			// another checkout. The main checkout must not mask its linked child.
			if current == nil || len(path) > len(resolvedPath(current.Path)) {
				current = &wts[i]
			}
		}
	}
	if current == nil || resolvedPath(current.Path) == resolvedPath(w.Root) {
		return w.Root, false, nil
	}
	if current.Branch != wantBranch {
		return "", false, clikit.Refusedf("current worktree is on %s, not task %s branch %s — run `dacli spawn --task %s --worktree` from the main checkout", current.Branch, t.ID, wantBranch, t.ID)
	}
	return filepath.Clean(current.Path), true, nil
}

// Git canonicalizes macOS's /tmp symlink to /private/tmp in porcelain output.
// Resolve both sides before containment checks or a real linked worktree can
// look unrelated solely because the caller used the shorter spelling.
func resolvedPath(path string) string {
	clean := filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(clean); err == nil {
		return filepath.Clean(resolved)
	}
	return clean
}

// hasAnyTrunk reports whether a conventional trunk branch exists at all.
func hasAnyTrunk(root string) bool {
	for _, b := range []string{"main", "master"} {
		if gitx.BranchExists(root, b) {
			return true
		}
	}
	return false
}

// cmdRuntimeRm is the removal inverse of the corresponding add. Every creatable object
// used to need a text editor to undo, which made a mistake a command made into
// a mistake only a human could fix (task 293).
func cmdRuntimeRm(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject(); err != nil {
		return err
	}
	if len(f.Pos) < 1 {
		return clikit.Usagef("usage: dacli runtime rm <name>")
	}
	name := f.Pos[0]
	if err := store.RemoveRuntime(w, name); err != nil {
		var ref store.ErrReferenced
		if errors.As(err, &ref) {
			// A dangling reference is worse than the mistake being removed, so
			// name what still points here rather than deleting and letting it
			// fail later at spawn time.
			return clikit.Refusedf("%v — retire or repoint them first", ref)
		}
		return err
	}
	fmt.Fprintf(ctx.Stdout, "removed runtime %s\n", name)
	return nil
}
