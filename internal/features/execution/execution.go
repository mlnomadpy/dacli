// Package execution is the runtime slice: dacli launching agents. Adapter
// management, single spawns, the supervision loop, and run records. This is
// the one slice that runs processes — and where the permission model stops
// being cooperative for spawned children (RUNTIMES.md § 8).
package execution

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "runtime add", Brief: "Add a coding-agent CLI adapter (--preset claude-code|generic-exec)", Run: cmdRuntimeAdd},
	{Path: "runtime list", Brief: "Configured runtimes and their declared capabilities", Run: cmdRuntimeList},
	{Path: "runtime doctor", Brief: "Probe installs: binary, version; declared-vs-probed kept distinct", Run: cmdRuntimeDoctor},
	{Path: "spawn", Brief: "Launch a child agent on a runtime: identity, brief, sandbox, run record", Run: cmdSpawn},
	{Path: "supervise", Brief: "Spawn-evaluate-correct loop until accepted or --max-turns", Run: cmdSupervise},
	{Path: "runs list", Brief: "Recorded agent runs, newest first", Run: cmdRunsList},
	{Path: "runs show", Brief: "Invocation, outcome, brief, and transcript for one run", Run: cmdRunsShow},
	{Path: "runs prune", Brief: "Bound transcript growth (--keep N, default 20)", Run: cmdRunsPrune},
	{Path: "agents", Brief: "Live spawned agents and the RAM/CPU/GPU their process trees hold", Run: cmdAgents},
	{Path: "kill", Brief: "Terminate an agent and its ENTIRE process tree (SIGTERM→SIGKILL); reaps runaways", Run: cmdKill},
}

// presets are shipped adapters. Their flags are ASSUMPTIONS, recorded as
// such in the adapter body, to be corrected by `runtime doctor` on a machine
// where the binary exists.
var presets = map[string]store.Runtime{
	"claude-code": {
		Name: "claude-code", Binary: "claude", Mode: "arg", Flag: "-p",
		// Read-only means read tools plus Bash scoped to the dacli binary —
		// plan mode would mute the child entirely (no Bash = no reporting).
		SandboxRO: []string{"--allowedTools", "Read,Grep,Glob,LS,Bash(dacli:*)"},
		// Deliberately NO ANTHROPIC_API_KEY: children run as the user's own
		// Claude Code login (keychain via HOME/USER), never API billing. If
		// that variable leaked through, billing would silently flip.
		Env:             []string{"HOME", "PATH", "USER", "LOGNAME", "TMPDIR"},
		ModelFlag:       "--model", // role-level cost routing: reviewer=opus, junior=haiku
		SkillsNativeDir: ".claude/skills",
	},
	"generic-exec": {
		Name: "generic-exec", Binary: "", Mode: "stdin",
		Env: []string{"HOME", "PATH"},
	},
}

func cmdRuntimeAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli runtime add <name> [--preset claude-code|generic-exec] [--binary b] [--mode stdin|arg] [--flag -p] [--arg a]... [--sandbox-ro-arg a]... [--env NAME]... [--model-flag f]\n(values that start with -- need the = form: --model-flag=--model)")
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
		rt.Env = v
	}
	if v := f.Get("model-flag"); v != "" {
		rt.ModelFlag = v
	}
	if v := f.Get("skills-native-dir"); v != "" {
		rt.SkillsNativeDir = v
	}
	if v := f.Get("skills-context-file"); v != "" {
		rt.SkillsContextFile = v
	}
	if err := store.CreateRuntime(w, id.ID, rt, ""); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "runtime %s added (binary: %s, mode: %s) — run `dacli runtime doctor` to probe it\n", rt.Name, rt.Binary, rt.Mode)
	return nil
}

func cmdRuntimeList(ctx *clikit.Ctx, args []string) error {
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
		fmt.Fprintf(ctx.Stdout, "%-14s %-16s %-6s %s\n", rt.Name, rt.Binary, rt.Mode, sandbox)
	}
	return nil
}

// cmdRuntimeDoctor probes what can be probed for free: binary on PATH and a
// --version call. Declared capabilities it cannot verify are reported as
// declared, never claimed.
func cmdRuntimeDoctor(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
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

// cmdSpawn launches a child agent: identity minted, brief assembled and
// FROZEN (the P3 replay capture), sandbox flags applied, process run to
// completion, everything written to a run record. Single-turn by design —
// the small-task assumption is the design center.
func cmdSpawn(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	taskRef := f.Get("task")
	if taskRef == "" {
		return clikit.Usagef("usage: dacli spawn --task <ref> [--runtime name] [--role r] [--grant ro|rw] [--model m] [--pr] [--review [--pr-number N]] [--budget N] [--timeout sec] [--cooperative]")
	}
	t, err := store.FindTask(w, taskRef)
	if err != nil {
		return err
	}

	// Role defaults: grant, runtime routing, model tier, WIP, seniority.
	grant := model.Grant(f.Get("grant"))
	roleName := f.Get("role")
	rtName := f.Get("runtime")
	modelName := f.Get("model")
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
				return clikit.Refusedf("role %s is at its WIP limit (%d/%d)", roleName, active, role.WIP)
			}
		}
		if err := seniorityGate(role, t); err != nil {
			return err
		}
		if err := phaseGate(w, t, role); err != nil {
			return err
		}
	}
	if grant == "" {
		grant = model.GrantRO
	}
	if rtName == "" {
		return clikit.Usagef("no runtime: pass --runtime or set `runtime:` on the role")
	}
	rt, err := store.LoadRuntime(w, rtName)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath(rt.Binary); err != nil {
		return fmt.Errorf("runtime %s: binary %q not on PATH — `dacli runtime doctor`", rt.Name, rt.Binary)
	}

	sandboxArgs, err := sandboxFor(ctx, rt, grant, f.Bool("cooperative"))
	if err != nil {
		return err
	}

	childID, token, err := agentid.Spawn(w, id, roleName, grant)
	if err != nil {
		return err
	}

	budget, _ := strconv.Atoi(f.Get("budget"))
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
	if n, _ := strconv.Atoi(f.Get("timeout")); n > 0 {
		timeout = n
	}

	invocation := fmt.Sprintf("run: %s\ntask: %s\nchild: %s\nrole: %s\ngrant: %s\nruntime: %s\nbinary: %s\nenv_names: %s\nbudget: %d (recorded, not enforced: runtime reports no usage)\ntimeout_s: %d\n",
		runID, t.ID, childID, clikit.OrDash(roleName), grant, rt.Name, rt.Binary,
		strings.Join(append([]string{agentid.EnvVar}, rt.Env...), ","), budget, timeout)
	writeRun("invocation.txt", invocation)

	// --worktree isolates this child in its own git worktree + branch, so
	// several children spawned in parallel never clobber each other's working
	// tree. The child works there; its branch is merged later via dacli merge.
	workDir := w.Root
	if f.Bool("worktree") {
		if !gitx.Available() {
			return fmt.Errorf("--worktree needs git on PATH")
		}
		wtPath := w.WorktreePath(t.Slug)
		if err := gitx.AddWorktree(w.Root, wtPath, fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug)); err != nil {
			// An existing worktree (a re-spawn) is fine; a real failure is not.
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
		}
		workDir = wtPath
		writeRun("worktree.txt", wtPath+"\n")
		fmt.Fprintf(ctx.Stderr, "isolated worktree: %s\n", wtPath)
	}

	extraArgs := append(append([]string{}, sandboxArgs...), modelArgs(ctx, rt, modelName)...)
	fmt.Fprintf(ctx.Stderr, "spawning %s on %s for %03d-%s (run %s)\n", childID, rt.Name, t.Seq, t.Slug, runID[:10])
	// Register the live process tree so `dacli agents`/`dacli kill` (a separate
	// invocation) can find and reap it while this spawn blocks here.
	onStart := func(pid, pgid int) {
		_ = procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), procmon.Record{
			RunID: runID, Child: childID, Task: t.ID, Role: roleName, Runtime: rt.Name,
			PID: pid, PGID: pgid, Started: time.Now(),
		})
	}
	outBytes, elapsed, timedOut, runErr := execRuntime(workDir, rt, prompt, token, extraArgs, timeout, onStart)
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
		outcome, clikit.ErrStr(runErr), elapsed, done, total, len(childEvents)))

	fmt.Fprintf(ctx.Stdout, "run %s: %s in %s · child wrote %d event(s) · acceptance %d/%d\ntranscript: %s\n",
		runID[:10], outcome, elapsed, len(childEvents), done, total, filepath.Join(runDir, "transcript.log"))
	if outcome == "failed" || outcome == "stalled" {
		return fmt.Errorf("child %s: %s (see %s)", childID, outcome, runDir)
	}
	return nil
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
	taskRef, rtName := f.Get("task"), f.Get("runtime")
	if taskRef == "" {
		return clikit.Usagef("usage: dacli supervise --task <ref> [--runtime name] [--role r] [--max-turns N] [--grant ro|rw] [--model m] [--pr] [--budget N] [--timeout sec] [--cooperative]")
	}
	t, err := store.FindTask(w, taskRef)
	if err != nil {
		return err
	}

	grant := model.Grant(f.Get("grant"))
	modelName := f.Get("model")
	if role, ok := store.LoadRole(w, f.Get("role")); ok {
		if grant == "" && role.Grant != "" {
			grant = model.Grant(role.Grant)
		}
		if rtName == "" {
			rtName = role.Runtime
		}
		if modelName == "" {
			modelName = role.Model
		}
		if err := seniorityGate(role, t); err != nil {
			return err
		}
		if err := phaseGate(w, t, role); err != nil {
			return err
		}
	}
	if grant == "" {
		grant = model.GrantRO
	}
	if rtName == "" {
		return clikit.Usagef("no runtime: pass --runtime or set `runtime:` on the role")
	}
	rt, err := store.LoadRuntime(w, rtName)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath(rt.Binary); err != nil {
		return fmt.Errorf("runtime %s: binary %q not on PATH", rt.Name, rt.Binary)
	}
	sandboxArgs, err := sandboxFor(ctx, rt, grant, f.Bool("cooperative"))
	if err != nil {
		return err
	}
	maxTurns := 3
	if n, _ := strconv.Atoi(f.Get("max-turns")); n > 0 {
		maxTurns = n
	}
	timeout := 300
	if n, _ := strconv.Atoi(f.Get("timeout")); n > 0 {
		timeout = n
	}
	budget, _ := strconv.Atoi(f.Get("budget"))

	// ONE child identity across turns: ownership continuity is what lets the
	// child claim on turn 1 and check its own boxes on turn 2.
	childID, token, err := agentid.Spawn(w, id, f.Get("role"), grant)
	if err != nil {
		return err
	}

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
		b, err := brief.Assemble(w, taskRef, brief.Options{Budget: budget})
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
		_ = os.WriteFile(filepath.Join(runDir, "brief.md"), []byte(prompt), 0o644)
		_ = os.WriteFile(filepath.Join(runDir, "invocation.txt"),
			[]byte(fmt.Sprintf("run: %s\nsupervise_turn: %d/%d\ntask: %s\nchild: %s\nruntime: %s\n", runID, turn, maxTurns, t.ID, childID, rt.Name)), 0o644)

		fmt.Fprintf(ctx.Stderr, "turn %d/%d: %s on %s\n", turn, maxTurns, childID, rt.Name)
		extraArgs := append(append([]string{}, sandboxArgs...), modelArgs(ctx, rt, modelName)...)
		onStart := func(pid, pgid int) {
			_ = procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), procmon.Record{
				RunID: runID, Child: childID, Task: t.ID, Role: f.Get("role"), Runtime: rt.Name,
				PID: pid, PGID: pgid, Started: time.Now(),
			})
		}
		out, elapsed, timedOut, runErr := execRuntime(w.Root, rt, prompt, token, extraArgs, timeout, onStart)
		_ = os.WriteFile(filepath.Join(runDir, "transcript.log"), out, 0o644)

		// The supervisor owns the objects, so it applies the child's events
		// between turns — claims become ownership, findings become notes.
		if res, err := eventlog.Sync(w, id.ID, id.CanMutate); err == nil && res.Applied > 0 {
			fmt.Fprintf(ctx.Stderr, "  applied %d child event(s)\n", res.Applied)
		}

		unmet := unmetList()
		cur, _ := store.FindTask(w, taskRef)
		outcome := fmt.Sprintf("turn %d: %d unmet, elapsed %s", turn, len(unmet), elapsed)
		_ = os.WriteFile(filepath.Join(runDir, "outcome.md"), []byte(outcome+"\n"), 0o644)

		if len(unmet) == 0 && cur != nil && cur.Status == model.StatusDone {
			fmt.Fprintf(ctx.Stdout, "accepted after %d turn(s): all acceptance criteria met and task done\n", turn)
			return nil
		}
		if timedOut {
			return fmt.Errorf("stalled: turn %d timed out after %ds (run %s)", turn, timeout, runID[:10])
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
	ph, err := gates.PhaseFor(w, t.Project)
	if err != nil || !ph.Gated {
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
func seniorityGate(role team.Role, t *store.Task) error {
	if role.MaxPoints <= 0 {
		return nil
	}
	tp, ok := t.Estimate()
	if !ok {
		return clikit.Refusedf("role %s takes only estimated tasks (max %g points) — estimate %03d-%s first", role.Name, role.MaxPoints, t.Seq, t.Slug)
	}
	if te := tp.Expected(); te > role.MaxPoints {
		return clikit.Refusedf("task %03d-%s is Te %.1f, above role %s's cap of %g — assign a heavier role, or decompose the task", t.Seq, t.Slug, te, role.Name, role.MaxPoints)
	}
	return nil
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
		return nil, nil
	}
	if len(rt.SandboxRO) > 0 {
		return rt.SandboxRO, nil
	}
	if !cooperative {
		return nil, clikit.Refusedf("runtime %s cannot enforce read-only; spawning an unrestricted process labeled ro would be a lie. Pass --cooperative to accept convention-only permissions, or use an rw grant", rt.Name)
	}
	fmt.Fprintf(ctx.Stderr, "warning: read-only is COOPERATIVE on %s — the child can bypass dacli; you accepted this with --cooperative\n", rt.Name)
	return nil, nil
}

// execRuntime launches one child turn. Env is allowlisted by NAME — the
// child gets the token plus exactly what the adapter declares, never the
// parent's full environment.
//
// The child is placed in its own process GROUP (Setpgid) so its entire
// subprocess tree can be signalled as a unit: on timeout we kill the group,
// not just the leader, and `dacli kill` can reap it later. onStart, if set, is
// called once the process exists with its (pid, pgid) so the caller can record
// a live-agent entry that a *separate* dacli invocation can find and kill.
func execRuntime(dir string, rt store.Runtime, prompt, token string, extraArgs []string, timeoutSec int, onStart func(pid, pgid int)) (out []byte, elapsed time.Duration, timedOut bool, err error) {
	argv := append([]string{}, rt.Args...)
	argv = append(argv, extraArgs...)
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
	// New process group: the child becomes group leader (pgid == its pid), and
	// every subprocess it forks inherits the group unless it detaches.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// On timeout/cancel, SIGKILL the whole GROUP. The default CommandContext
	// cancel kills only the leader — which would orphan the children the agent
	// spawned, exactly the runaway leak we are preventing.
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	// Bound how long Wait blocks on output pipes a grandchild may still hold
	// open after the group was killed, so a hung tree can't wedge dacli.
	cmd.WaitDelay = 5 * time.Second

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if rt.Mode == "stdin" {
		cmd.Stdin = strings.NewReader(prompt)
	}
	start := time.Now()
	if serr := cmd.Start(); serr != nil {
		return buf.Bytes(), time.Since(start).Round(time.Millisecond), false, serr
	}
	if onStart != nil {
		onStart(cmd.Process.Pid, cmd.Process.Pid) // pgid == leader pid under Setpgid
	}
	err = cmd.Wait()
	return buf.Bytes(), time.Since(start).Round(time.Millisecond), cctx.Err() == context.DeadlineExceeded, err
}

// promptSuffix assembles everything appended after the brief: the reporting
// protocol, git discipline for writers, review discipline for reviewers.
// All of it lives in the prompt registry, none of it in Fprintf chains.
func promptSuffix(w *workspace.Workspace, f *clikit.Flags, t *store.Task, childID string, grant model.Grant) (string, error) {
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
			"Ref":    fmt.Sprintf("%03d", t.Seq),
			"Title":  t.Title,
			"Branch": fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug),
			"PR":     f.Bool("pr"),
			"Exe":    exe,
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
			"Search": fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug),
			"PRRef":  f.Get("pr-number"),
			"Exe":    exe,
		})
		if err != nil {
			return "", err
		}
		out += "\n" + review
	}
	return out, nil
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

func cmdRunsList(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
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

func cmdRunsShow(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli runs show <run-id-prefix>")
	}
	entries, _ := os.ReadDir(w.RunsDir())
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), f.Pos[0]) {
			continue
		}
		for _, name := range []string{"invocation.txt", "outcome.md", "brief.md", "transcript.log"} {
			if raw, err := os.ReadFile(filepath.Join(w.RunDir(e.Name()), name)); err == nil {
				fmt.Fprintf(ctx.Stdout, "=== %s ===\n%s\n", name, strings.TrimSpace(string(raw)))
			}
		}
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
	keep := 20
	if n, _ := strconv.Atoi(f.Get("keep")); n > 0 {
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

// cmdAgents lists agents whose process tree is still alive, with the RAM/CPU
// (and GPU where measurable) the whole group is holding right now. A run's
// proc.txt is written at spawn; liveness is probed live, so an exited agent
// simply doesn't appear — the list is runaways-included, ghosts-excluded.
func cmdAgents(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	for _, rec := range liveAgents(w) {
		u := procmon.SampleGroup(rec.PGID)
		age := time.Since(rec.Started).Round(time.Second)
		fmt.Fprintf(ctx.Stdout, "%s  %-14s %-12s %-10s pid %-7d %2d proc  %8s RAM  %5.0f%% CPU  %7s GPU  up %s\n",
			rec.RunID[:min(10, len(rec.RunID))], clikit.OrDash(rec.Child), clikit.OrDash(rec.Runtime),
			"task "+clikit.OrDash(rec.Task), rec.PID, u.Procs, humanKB(u.RSSKB), u.CPUPct, gpuStr(u.GPUMiB), age)
	}
	if len(liveAgents(w)) == 0 {
		fmt.Fprintln(ctx.Stdout, "no live agents")
	}
	return nil
}

// cmdKill terminates one agent's whole process tree, or --all of them. The
// group is SIGTERM'd, then SIGKILL'd after a grace window if anything survives
// — so a well-behaved agent exits cleanly and a hung one is still guaranteed
// dead, with no orphaned children left holding resources.
func cmdKill(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	grace := time.Duration(3) * time.Second
	if n, _ := strconv.Atoi(f.Get("grace")); n > 0 {
		grace = time.Duration(n) * time.Second
	}

	if f.Bool("all") {
		live := liveAgents(w)
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
	for _, rec := range liveAgents(w) {
		if strings.HasPrefix(rec.RunID, ref) || rec.Child == ref {
			killOne(ctx, w, rec, grace)
			return nil
		}
	}
	return store.ErrNotFound{Ref: "live agent " + ref}
}

// liveAgents reads every run's proc.txt and returns those whose leader process
// is still alive, newest first.
func liveAgents(w *workspace.Workspace) []procmon.Record {
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
	var out []procmon.Record
	for _, n := range names {
		rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(n), "proc.txt"))
		if err != nil {
			continue
		}
		if procmon.Alive(rec.PID) {
			out = append(out, rec)
		}
	}
	return out
}

func killOne(ctx *clikit.Ctx, w *workspace.Workspace, rec procmon.Record, grace time.Duration) {
	before := procmon.SampleGroup(rec.PGID).Procs
	termed, killed := procmon.KillTree(rec.PGID, grace)
	verb := "SIGTERM"
	if killed {
		verb = "SIGTERM→SIGKILL"
	}
	if !termed && !killed {
		fmt.Fprintf(ctx.Stdout, "%s: nothing to signal (already gone)\n", clikit.OrDash(rec.Child))
		return
	}
	// Audit crumb next to the run record: what was reaped and how.
	_ = os.WriteFile(filepath.Join(w.RunDir(rec.RunID), "killed.txt"),
		[]byte(fmt.Sprintf("killed %s (pgid %d, ~%d proc) via %s at %s\n",
			rec.Child, rec.PGID, before, verb, time.Now().UTC().Format(time.RFC3339))), 0o644)
	fmt.Fprintf(ctx.Stdout, "killed %s — process group %d (~%d proc) reaped via %s\n",
		clikit.OrDash(rec.Child), rec.PGID, before, verb)
}

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
