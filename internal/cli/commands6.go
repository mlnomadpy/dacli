// Sixth slice: runtimes — dacli launches the agents itself. This is where
// the permission model stops being cooperative for spawned children: dacli
// sets the runtime's sandbox flags, and a runtime that cannot enforce
// read-only causes a refusal, never a silent downgrade (RUNTIMES.md § 8).
package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// presets are shipped adapters. Their flags are ASSUMPTIONS, recorded as
// such in the adapter body, to be corrected by `runtime doctor` on a machine
// where the binary exists.
var presets = map[string]store.Runtime{
	"claude-code": {
		Name: "claude-code", Binary: "claude", Mode: "arg", Flag: "-p",
		// Read-only means read tools plus Bash scoped to the dacli binary —
		// plan mode would mute the child entirely (no Bash = no reporting).
		// Workspaces override the binary path in the pattern after add.
		SandboxRO: []string{"--allowedTools", "Read,Grep,Glob,LS,Bash(dacli:*)"},
		// Deliberately NO ANTHROPIC_API_KEY: children run as the user's own
		// Claude Code login (keychain via HOME/USER), never API billing. If
		// that variable leaked through, billing would silently flip.
		Env:       []string{"HOME", "PATH", "USER", "LOGNAME", "TMPDIR"},
		ModelFlag: "--model", // role-level cost routing: reviewer=opus, junior=haiku
	},
	"generic-exec": {
		Name: "generic-exec", Binary: "", Mode: "stdin",
		Env: []string{"HOME", "PATH"},
	},
}

func cmdRuntimeAdd(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli runtime add <name> [--preset claude-code|generic-exec] [--binary b] [--mode stdin|arg] [--flag -p] [--arg a]... [--sandbox-ro-arg a]... [--env NAME]... [--model-flag f]\n(values that start with -- need the = form: --model-flag=--model)")
	}
	rt := store.Runtime{Name: f.pos[0]}
	if p := f.get("preset"); p != "" {
		base, ok := presets[p]
		if !ok {
			return usagef("unknown preset %q", p)
		}
		base.Name = rt.Name
		rt = base
	}
	if v := f.get("binary"); v != "" {
		rt.Binary = v
	}
	if v := f.get("mode"); v != "" {
		rt.Mode = v
	}
	if v := f.get("flag"); v != "" {
		rt.Flag = v
	}
	if v := f.all("arg"); len(v) > 0 {
		rt.Args = v
	}
	if v := f.all("sandbox-ro-arg"); len(v) > 0 {
		rt.SandboxRO = v
	}
	if v := f.all("env"); len(v) > 0 {
		rt.Env = v
	}
	if v := f.get("model-flag"); v != "" {
		rt.ModelFlag = v
	}
	if err := store.CreateRuntime(w, id.ID, rt, ""); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "runtime %s added (binary: %s, mode: %s) — run `dacli runtime doctor` to probe it\n", rt.Name, rt.Binary, rt.Mode)
	return nil
}

func cmdRuntimeList(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	rts, _ := store.LoadRuntimes(w)
	for _, rt := range rts {
		sandbox := "no read-only mode"
		if len(rt.SandboxRO) > 0 {
			sandbox = "ro: " + strings.Join(rt.SandboxRO, " ")
		}
		fmt.Fprintf(ctx.Stdout, "%-14s %-16s %-6s %s\n", rt.Name, rt.Binary, rt.Mode, sandbox)
	}
	return nil
}

// cmdRuntimeDoctor probes what can be probed for free: binary on PATH and a
// --version call. Declared capabilities it cannot verify are reported as
// declared, never claimed — the ✓/✗/declared distinction is the point.
func cmdRuntimeDoctor(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	rts, _ := store.LoadRuntimes(w)
	if len(rts) == 0 {
		fmt.Fprintln(ctx.Stdout, "no runtimes configured; `dacli runtime add <name> --preset ...`")
		return nil
	}
	for _, rt := range rts {
		path, lerr := exec.LookPath(rt.Binary)
		if lerr != nil {
			fmt.Fprintf(ctx.Stdout, "%-14s ✗ binary %q not found on PATH\n", rt.Name, rt.Binary)
			continue
		}
		version := "version unknown"
		cctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if out, verr := exec.CommandContext(cctx, path, "--version").CombinedOutput(); verr == nil {
			version = strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
			if len(version) > 40 {
				version = version[:40]
			}
		}
		cancel()
		sandbox := "✗ no read-only mode (ro spawns will be refused)"
		if len(rt.SandboxRO) > 0 {
			sandbox = "sandbox declared (unprobed — probing would cost a real call)"
		}
		fmt.Fprintf(ctx.Stdout, "%-14s ✓ %s · %s · %s\n", rt.Name, path, version, sandbox)
	}
	return nil
}

// cmdSpawn launches a child agent on a runtime: identity minted, brief
// assembled and recorded, sandbox flags applied, process run to completion,
// everything written to a run record. Single-turn by design — the small-task
// assumption is the design center, and turn 3 means the task was mis-sized.
func cmdSpawn(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	taskRef := f.get("task")
	if taskRef == "" {
		return usagef("usage: dacli spawn --task <ref> [--runtime name] [--role r] [--grant ro|rw] [--model m] [--pr] [--review [--pr-number N]] [--budget N] [--timeout sec] [--cooperative]")
	}
	t, err := store.FindTask(w, taskRef)
	if err != nil {
		return err
	}

	// Role defaults: grant, runtime routing, model tier, WIP, seniority.
	grant := model.Grant(f.get("grant"))
	roleName := f.get("role")
	rtName := f.get("runtime")
	modelName := f.get("model")
	if role, ok := store.LoadRole(w, roleName); ok {
		if grant == "" && role.Grant != "" {
			grant = model.Grant(role.Grant)
		}
		if rtName == "" {
			rtName = role.Runtime
		}
		if modelName == "" {
			modelName = role.Model
		}
		if role.WIP > 0 {
			if active := store.ActiveInRole(w, roleName); active >= role.WIP {
				return refusedf("role %s is at its WIP limit (%d/%d)", roleName, active, role.WIP)
			}
		}
		if err := seniorityGate(role, t); err != nil {
			return err
		}
	}
	if grant == "" {
		grant = model.GrantRO
	}
	if rtName == "" {
		return usagef("no runtime: pass --runtime or set `runtime:` on the role")
	}
	rt, err := store.LoadRuntime(w, rtName)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath(rt.Binary); err != nil {
		return fmt.Errorf("runtime %s: binary %q not on PATH — `dacli runtime doctor`", rt.Name, rt.Binary)
	}

	sandboxArgs, err := sandboxFor(ctx, rt, grant, f.bool("cooperative"))
	if err != nil {
		return err
	}

	childID, token, err := agentid.Spawn(w, id, roleName, grant)
	if err != nil {
		return err
	}

	budget, _ := strconv.Atoi(f.get("budget"))
	b, err := brief.Assemble(w, taskRef, brief.Options{Budget: budget})
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
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	writeRun := func(name, content string) {
		_ = os.WriteFile(filepath.Join(runDir, name), []byte(content), 0o644)
	}
	writeRun("brief.md", prompt)

	timeout := 300
	if n, _ := strconv.Atoi(f.get("timeout")); n > 0 {
		timeout = n
	}

	invocation := fmt.Sprintf("run: %s\ntask: %s\nchild: %s\nrole: %s\ngrant: %s\nruntime: %s\nbinary: %s\nenv_names: %s\nbudget: %d (recorded, not enforced: runtime reports no usage)\ntimeout_s: %d\n",
		runID, t.ID, childID, orDash(roleName), grant, rt.Name, rt.Binary,
		strings.Join(append([]string{agentid.EnvVar}, rt.Env...), ","), budget, timeout)
	writeRun("invocation.txt", invocation)

	extraArgs := append(append([]string{}, sandboxArgs...), modelArgs(ctx, rt, modelName)...)
	fmt.Fprintf(ctx.Stderr, "spawning %s on %s for %03d-%s (run %s)\n", childID, rt.Name, t.Seq, t.Slug, runID[:10])
	outBytes, elapsed, timedOut, runErr := execRuntime(w.Root, rt, prompt, token, extraArgs, timeout)
	writeRun("transcript.log", string(outBytes))

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
	childEvents, _ := eventlog.List(w, eventlog.Query{Actor: childID})

	outcome := "ok"
	switch {
	case timedOut:
		outcome = "stalled"
	case runErr != nil && len(childEvents) > 0:
		outcome = "partial"
	case runErr != nil:
		outcome = "failed"
	}
	writeRun("outcome.md", fmt.Sprintf("outcome: %s\nexit: %v\nelapsed: %s\nacceptance: %d/%d\nevents_by_child: %d\n",
		outcome, errStr(runErr), elapsed, done, total, len(childEvents)))

	fmt.Fprintf(ctx.Stdout, "run %s: %s in %s · child wrote %d event(s) · acceptance %d/%d\ntranscript: %s\n",
		runID[:10], outcome, elapsed, len(childEvents), done, total, filepath.Join(runDir, "transcript.log"))
	if outcome == "failed" || outcome == "stalled" {
		return fmt.Errorf("child %s: %s (see %s)", childID, outcome, runDir)
	}
	return nil
}

// sandboxFor applies the § 8 rule: a read-only child needs a runtime that
// can enforce it. --cooperative downgrades EXPLICITLY and loudly; silence is
// the failure mode this rule exists to prevent.
func sandboxFor(ctx *Ctx, rt store.Runtime, grant model.Grant, cooperative bool) ([]string, error) {
	if grant != model.GrantRO {
		return nil, nil
	}
	if len(rt.SandboxRO) > 0 {
		return rt.SandboxRO, nil
	}
	if !cooperative {
		return nil, refusedf("runtime %s cannot enforce read-only; spawning an unrestricted process labeled ro would be a lie. Pass --cooperative to accept convention-only permissions, or use an rw grant", rt.Name)
	}
	fmt.Fprintf(ctx.Stderr, "warning: read-only is COOPERATIVE on %s — the child can bypass dacli; you accepted this with --cooperative\n", rt.Name)
	return nil, nil
}

// execRuntime launches one child turn. Env is allowlisted by NAME — the
// child gets the token plus exactly what the adapter declares, never the
// parent's full environment.
func execRuntime(dir string, rt store.Runtime, prompt, token string, sandboxArgs []string, timeoutSec int) (out []byte, elapsed time.Duration, timedOut bool, err error) {
	argv := append([]string{}, rt.Args...)
	argv = append(argv, sandboxArgs...)
	if rt.Mode == "arg" {
		if rt.Flag != "" {
			argv = append(argv, rt.Flag)
		}
		argv = append(argv, prompt)
	}
	env := []string{agentid.EnvVar + "=" + token}
	for _, name := range rt.Env {
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}

	cctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, rt.Binary, argv...)
	cmd.Dir = dir
	cmd.Env = env
	if rt.Mode == "stdin" {
		cmd.Stdin = strings.NewReader(prompt)
	}
	start := time.Now()
	out, err = cmd.CombinedOutput()
	return out, time.Since(start).Round(time.Millisecond), cctx.Err() == context.DeadlineExceeded, err
}

func errStr(err error) string {
	if err == nil {
		return "0"
	}
	return err.Error()
}

// protocolPreamble tells a spawned child HOW to report. Without it, a real
// headless child does the work and prints text into the void — work not
// written to the workspace does not exist.
//
// The prose lives in internal/prompts/tpl/protocol_preamble.md (workspace-
// overridable via .dacli/prompts/), not here: prompts are data, and a prompt
// in an Fprintf chain cannot be audited or improved without a recompile.
// seniorityGate enforces a role's MaxPoints: a junior role mechanically
// cannot take the hard migration. Unestimated tasks are refused too — a
// capped role takes only work whose size somebody stated.
func seniorityGate(role team.Role, t *store.Task) error {
	if role.MaxPoints <= 0 {
		return nil
	}
	tp, ok := t.Estimate()
	if !ok {
		return refusedf("role %s takes only estimated tasks (max %g points) — estimate %03d-%s first", role.Name, role.MaxPoints, t.Seq, t.Slug)
	}
	if te := tp.Expected(); te > role.MaxPoints {
		return refusedf("task %03d-%s is Te %.1f, above role %s's cap of %g — assign a heavier role, or decompose the task", t.Seq, t.Slug, te, role.Name, role.MaxPoints)
	}
	return nil
}

// modelArgs routes a model tier onto the runtime. A runtime with no model
// flag makes role-level routing inoperative — announced, never ignored: a
// junior role silently running on the expensive default is exactly the cost
// leak the routing exists to prevent.
func modelArgs(ctx *Ctx, rt store.Runtime, modelName string) []string {
	if modelName == "" {
		return nil
	}
	if rt.ModelFlag == "" {
		fmt.Fprintf(ctx.Stderr, "warning: model %q requested but runtime %s declares no model_flag — running on the runtime's default\n", modelName, rt.Name)
		return nil
	}
	return []string{rt.ModelFlag, modelName}
}

// promptSuffix assembles everything appended after the brief: the reporting
// protocol, git discipline for writers, review discipline for reviewers.
// All of it lives in the prompt registry, none of it in Fprintf chains.
func promptSuffix(w *workspace.Workspace, f *flags, t *store.Task, childID string, grant model.Grant) (string, error) {
	out, err := protocolPreamble(w, childID, grant, t)
	if err != nil {
		return "", err
	}
	if grant == model.GrantRW {
		git, err := prompts.Render(w.PromptsDir(), "git_workflow", map[string]any{
			"Ref":    fmt.Sprintf("%03d", t.Seq),
			"Title":  t.Title,
			"Branch": fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug),
			"PR":     f.bool("pr"),
		})
		if err != nil {
			return "", err
		}
		out += "\n" + git
	}
	if f.bool("review") {
		review, err := prompts.Render(w.PromptsDir(), "review_workflow", map[string]any{
			"Search": t.ID,
			"PRRef":  f.get("pr-number"),
		})
		if err != nil {
			return "", err
		}
		out += "\n" + review
	}
	return out, nil
}

func protocolPreamble(w *workspace.Workspace, childID string, grant model.Grant, t *store.Task) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		exe = "dacli"
	}
	out, err := prompts.Render(w.PromptsDir(), "protocol_preamble", map[string]any{
		"ChildID": childID,
		"Grant":   string(grant),
		"Ref":     fmt.Sprintf("%03d", t.Seq),
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

func cmdRunsList(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return nil
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
		fmt.Fprintf(ctx.Stdout, "%s  %s\n", n[:10], line)
	}
	return nil
}

func cmdRunsShow(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli runs show <run-id-prefix>")
	}
	entries, _ := os.ReadDir(w.RunsDir())
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), f.pos[0]) {
			continue
		}
		for _, name := range []string{"invocation.txt", "outcome.md", "brief.md", "transcript.log"} {
			if raw, err := os.ReadFile(filepath.Join(w.RunDir(e.Name()), name)); err == nil {
				fmt.Fprintf(ctx.Stdout, "=== %s ===\n%s\n", name, strings.TrimSpace(string(raw)))
			}
		}
		return nil
	}
	return store.ErrNotFound{Ref: "run " + f.pos[0]}
}

func cmdRunsPrune(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	keep := 20
	if n, _ := strconv.Atoi(f.get("keep")); n > 0 {
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
	pruned := 0
	for len(names) > keep {
		if err := os.RemoveAll(w.RunDir(names[0])); err != nil {
			return err
		}
		names = names[1:]
		pruned++
	}
	fmt.Fprintf(ctx.Stdout, "pruned %d run(s), kept %d\n", pruned, len(names))
	return nil
}
