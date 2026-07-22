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
	"fmt"
	"io"
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
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "runtime add", Brief: "Add a coding-agent CLI adapter (--preset claude-code|generic-exec)", Run: cmdRuntimeAdd},
	{Path: "runtime list", Brief: "Configured runtimes and their declared capabilities", Run: cmdRuntimeList},
	{Path: "runtime doctor", Brief: "Probe installs: binary, version; declared-vs-probed kept distinct", Run: cmdRuntimeDoctor},
	{Path: "spawn", Brief: "Launch a child agent on a runtime: identity, brief, sandbox, run record (--detach to background)", Run: cmdSpawn},
	{Path: "wait", Brief: "Block until detached run(s) finish, then finalize their outcome (default: all live)", Run: cmdWait},
	{Path: "supervise", Brief: "Spawn-evaluate-correct loop until accepted or --max-turns", Run: cmdSupervise},
	{Path: "runs list", Brief: "Recorded agent runs, newest first", Run: cmdRunsList},
	{Path: "runs show", Brief: "Invocation, outcome, brief, and transcript for one run", Run: cmdRunsShow},
	{Path: "runs prune", Brief: "Bound transcript growth (--keep N, default 20)", Run: cmdRunsPrune},
	{Path: "agents", Brief: "Live spawned agents + RAM/CPU/GPU; --tail shows each one's last transcript line (thinking vs hung); --max-rss/--max-runtime --reap kills over-budget trees", Run: cmdAgents},
	{Path: "logs", Brief: "Print or follow (-f) a run's transcript as it streams", Run: cmdLogs},
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
	if v := f.Get("usage-format"); v != "" {
		rt.UsageFormat = v // F1 opt-in: "stream-json" captures token actuals
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
// claimTask stamps "claimed by <childID>" onto the task's Log at spawn so a
// claim->completed span exists for that task. calibration.logSpan reads the
// FIRST "claimed by" stamp as the span start (calibration.go:141); without it
// calibrate's by-agent band has no span to join run records against and stays
// empty on real runs (D1). Idempotent: only stamp when no claim exists yet, so
// a re-spawn or a multi-turn supervise respects the first owner and never adds
// a second claim (which would move the span start). The task is loaded from the
// shared root, so the stamp lands there and travels with the task.
func claimTask(ctx *clikit.Ctx, t *store.Task, childID string) {
	if s, found := t.Doc.Section("Log"); found && strings.Contains(s.Content, "claimed by") {
		return
	}
	store.AppendLog(t, "claimed by "+childID)
	if err := store.SaveTask(t); err != nil {
		fmt.Fprintf(ctx.Stderr, "warning: could not stamp claim on task %03d-%s: %v\n", t.Seq, t.Slug, err)
	}
}

func cmdSpawn(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	taskRef := f.Get("task")
	if taskRef == "" {
		return clikit.Usagef("usage: dacli spawn --task <ref> [--runtime name] [--role r] [--grant ro|rw] [--model m] [--worktree] [--detach] [--claim path,path] [--pr] [--review [--pr-number N]] [--budget N] [--max-tokens N] [--timeout sec] [--cooperative] [--advise] [--force]")
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

	// --advise (D2): with role/model/runtime/task now resolved but BEFORE any
	// identity is minted or process launched, surface what the log already knows
	// at the spawn decision — a calibrated sizing for this agent band and this
	// task's taint status — then continue the spawn unchanged. Advice is
	// additive; it never decides (axiom 3: the intelligence stays the model's).
	// The band is built in the SAME recorded form invocation.txt uses (OrDash
	// for an empty role/model, rt.Name for runtime) so it matches the bands
	// store.CalibrationSamples joins back from the run records.
	band := store.Band{Role: clikit.OrDash(roleName), Model: clikit.OrDash(modelName), Runtime: rt.Name}
	if f.Bool("advise") {
		printAdvisory(ctx, w, t, band)
	}

	// Spawn-time token GATE (F2): --advise DISPLAYS the suggested token budget;
	// --max-tokens N ENFORCES it. If this band's expected token cost (F1's
	// measured output-tokens/point × the task's Te) EXCEEDS N, refuse the spawn
	// (exit 3) unless --force — the D3 taint-gate shape applied to cost. Below the
	// n≥10 calibration gate the estimate is PROVISIONAL: warn but never hard-
	// refuse on thin data. A band with no token history (text runtime) or an
	// unestimated task has nothing to enforce honestly, so it proceeds with a note.
	if maxStr := f.Get("max-tokens"); maxStr != "" {
		maxTok, err := strconv.Atoi(maxStr)
		if err != nil || maxTok <= 0 {
			return clikit.Usagef("--max-tokens takes a positive integer")
		}
		expected, n, ok := bandTokenBudget(w, t, band)
		switch {
		case !ok:
			fmt.Fprintf(ctx.Stderr, "note: --max-tokens %d not enforced — band %s has no measured token cost yet (or task unestimated)\n", maxTok, band)
		case expected <= float64(maxTok):
			// within budget — nothing to say
		case n < 10:
			fmt.Fprintf(ctx.Stderr, "warning: band %s expected ~%.0f tokens exceeds --max-tokens %d, but the estimate is PROVISIONAL (n=%d < 10) — spawning anyway\n", band, expected, maxTok, n)
		case f.Bool("force"):
			fmt.Fprintf(ctx.Stderr, "warning: --force spawning over token budget — band %s expected ~%.0f tokens exceeds --max-tokens %d (n=%d)\n", band, expected, maxTok, n)
		default:
			return clikit.Refusedf("band %s expected ~%.0f tokens exceeds --max-tokens %d (n=%d calibrated samples) — re-run with --force to spawn anyway", band, expected, maxTok, n)
		}
	}

	// Spawn-time taint GATE (D3): --advise DISPLAYS taint status; here it BLOCKS.
	// If this task's brief sits in an external source's blast radius, refuse the
	// spawn (exit 3) rather than feed a possibly-injected brief to a fresh child
	// — taint stops being an audit query you run after the fact and becomes a
	// gate at the point of consumption (RUNTIMES §18, cross-tree injection).
	// --force (or --cooperative) is the explicit, loud override: the operator has
	// read the origins and accepts the risk.
	if origins, inRadius, _ := externalRadius(w, t); inRadius && !(f.Bool("force") || f.Bool("cooperative")) {
		return clikit.Refusedf("task %03d-%s is in the blast radius of %s — its brief may carry injected content (RUNTIMES §18); audit the origins, then re-run with --force to spawn anyway",
			t.Seq, t.Slug, strings.Join(origins, ", "))
	}

	sandboxArgs, err := sandboxFor(ctx, rt, grant, f.Bool("cooperative"))
	if err != nil {
		return err
	}

	// --claim declares the paths this agent will edit. If a live agent already
	// claims an overlapping tree, refuse — this is the disjointness that keeps
	// parallel branches merge-clean, enforced instead of hoped for.
	claims := splitClaims(f.Get("claim"))
	if len(claims) > 0 {
		for _, other := range liveAgents(w) {
			if mine, theirs, clash := procmon.PathsOverlap(claims, other.Claims); clash {
				return clikit.Refusedf("path-claim conflict: live agent %s already claims %q and you claim %q — narrow your scope, or `dacli wait %s` first",
					other.Child, theirs, mine, other.RunID[:min(10, len(other.RunID))])
			}
		}
	}

	childID, token, err := agentid.Spawn(w, id, roleName, grant)
	if err != nil {
		return err
	}
	// Stamp the claim now that the child id is minted: this is the span start
	// calibrate joins run actuals against (D1). Idempotent — a re-spawn respects
	// the existing claim.
	claimTask(ctx, t, childID)

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
		// Surface a failed run-record write: swallowing it here later surfaces
		// downstream as an unexplained "brief not recorded" with no hint a write
		// actually failed.
		if err := os.WriteFile(filepath.Join(runDir, name), []byte(content), 0o644); err != nil {
			fmt.Fprintf(ctx.Stderr, "warning: could not record %s for run %s: %v\n", name, runID[:10], err)
		}
	}
	writeRun("brief.md", prompt)

	timeout := 300
	if n, _ := strconv.Atoi(f.Get("timeout")); n > 0 {
		timeout = n
	}

	invocation := fmt.Sprintf("run: %s\ntask: %s\nchild: %s\nrole: %s\nmodel: %s\ngrant: %s\nruntime: %s\nbinary: %s\nenv_names: %s\nbudget: %d (recorded, not enforced: runtime reports no usage)\ntimeout_s: %d\n",
		runID, t.ID, childID, clikit.OrDash(roleName), clikit.OrDash(modelName), grant, rt.Name, rt.Binary,
		strings.Join(append([]string{agentid.EnvVar}, rt.Env...), ","), budget, timeout)
	writeRun("invocation.txt", invocation)

	// --worktree isolates this child in its own git worktree + branch, so
	// several children spawned in parallel never clobber each other's working
	// tree. The child edits CODE there; workspace.Find redirects its dacli
	// state (identity, tasks, events) to the shared root, so the child sees its
	// own freshly-minted identity and can self-commit, self-check, self-report
	// — no shadow .dacli. Its events therefore land in the shared root, which
	// is exactly where we read them back from below.
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
			PID: pid, PGID: pgid, PIDStart: pidStart(pid), Started: time.Now(), Claims: claims,
		})
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
		writeRun("outcome.md", fmt.Sprintf("outcome: running (detached)\nchild: %s\ntask: %s\n", childID, t.ID))
		fmt.Fprintf(ctx.Stdout, "detached %s on %s for %03d-%s (run %s)\ntrack: dacli agents · block: dacli wait %s · transcript: %s\n",
			childID, rt.Name, t.Seq, t.Slug, runID[:10], runID[:10], transcriptPath)
		return nil
	}

	elapsed, timedOut, runErr := execRuntime(workDir, transcriptPath, rt, prompt, token, extraArgs, timeout, false, onStart)

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
func printAdvisory(ctx *clikit.Ctx, w *workspace.Workspace, t *store.Task, band store.Band) {
	fmt.Fprintf(ctx.Stdout, "── advise · %03d-%s · band %s ──\n", t.Seq, t.Slug, band.String())

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
	fmt.Fprintln(ctx.Stdout, "── (advice only; the spawn proceeds unchanged) ──")
}

// bandTokenBudget computes the expected token cost of spawning THIS band on task
// t from the band's measured output-tokens/point (F1): expected = median
// TokenRatio × the task's Te. It reads the SAME calibration samples the advisory
// displays via store.MedianTokenRatio, so the `--max-tokens` gate and `--advise`
// can never disagree on the number. n is the count of token-bearing samples in
// the band (the caller warns rather than refuses when n < 10). ok is false —
// nothing to enforce honestly — when the band has no token history (a text
// runtime) or the task carries no three-point estimate.
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
	// One child owns this task across turns; claim once (idempotent) so a
	// claim->completed span exists for calibrate to join (D1).
	claimTask(ctx, t, childID)

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
				PID: pid, PGID: pgid, PIDStart: pidStart(pid), Started: time.Now(),
			})
		}
		elapsed, timedOut, runErr := execRuntime(w.Root, filepath.Join(runDir, "transcript.log"), rt, prompt, token, extraArgs, timeout, false, onStart)

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
//
// The child's stdout+stderr stream to transcriptPath (a real file), so a
// DETACHED child's output persists after this parent process exits: the child
// keeps its own inherited fd and the parent closes its copy. detach=true starts
// the child in its own process group, releases it, and returns immediately with
// no deadline (enforce timeouts via `dacli kill` or a watchdog); the foreground
// path keeps the context deadline and group-kill-on-timeout.
func execRuntime(dir, transcriptPath string, rt store.Runtime, prompt, token string, extraArgs []string, timeoutSec int, detach bool, onStart func(pid, pgid int)) (elapsed time.Duration, timedOut bool, err error) {
	argv := append([]string{}, rt.Args...)
	argv = append(argv, extraArgs...)
	// F1: opt-in usage capture. Only when the adapter sets usage_format do we
	// ask the child to emit a machine-readable event stream; an empty
	// UsageFormat leaves argv (and thus a text runtime) exactly as it was. The
	// claude CLI requires --verbose alongside stream-json under --print.
	streamJSON := rt.UsageFormat == "stream-json"
	if streamJSON {
		argv = append(argv, "--output-format", "stream-json", "--verbose")
	}
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
	var sink *os.File
	if transcriptPath != "" {
		sink, _ = os.Create(transcriptPath)
	}
	start := time.Now()

	if detach {
		// Detached: no CommandContext (its deadline would fire on the parent's
		// exit and kill the child). New process group so the tree stays killable
		// and survives this process as its own group; Release() hands it off.
		cmd := exec.Command(rt.Binary, argv...)
		cmd.Dir = dir
		cmd.Env = env
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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
			return 0, false, serr
		}
		if onStart != nil {
			onStart(cmd.Process.Pid, cmd.Process.Pid)
		}
		_ = cmd.Process.Release()
		return 0, false, nil
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
	// Bound how long Wait blocks on output a grandchild may still hold open
	// after the group was killed, so a hung tree can't wedge dacli.
	cmd.WaitDelay = 5 * time.Second

	// stream-json capture: read the child's stdout through a pipe, tee a
	// human-readable rendering into the transcript (so logs -f / --tail keep
	// working) and remember the final usage event. Text runtimes keep the raw
	// stdout+stderr → sink wiring exactly as before.
	var streamPipe io.ReadCloser
	if streamJSON && sink != nil {
		streamPipe, _ = cmd.StdoutPipe()
		cmd.Stderr = sink
		defer sink.Close()
	} else if sink != nil {
		cmd.Stdout, cmd.Stderr = sink, sink
		defer sink.Close()
	}
	if rt.Mode == "stdin" {
		cmd.Stdin = strings.NewReader(prompt)
	}
	if serr := cmd.Start(); serr != nil {
		return time.Since(start).Round(time.Millisecond), false, serr
	}
	if onStart != nil {
		onStart(cmd.Process.Pid, cmd.Process.Pid) // pgid == leader pid under Setpgid
	}
	if streamPipe != nil {
		// Must drain the pipe fully before Wait (os/exec closes it on exit).
		u := teeStreamJSON(streamPipe, sink)
		err = cmd.Wait()
		if u.found {
			writeUsage(filepath.Dir(transcriptPath), u)
		} else if u.scanErr != nil {
			// The stream ended before the result event: usage was lost. Make that
			// visible in the transcript instead of falling back to the wall-clock
			// proxy as if this were a plain text runtime.
			fmt.Fprintf(sink, "[dacli: usage capture incomplete — %v]\n", u.scanErr)
		}
		return time.Since(start).Round(time.Millisecond), cctx.Err() == context.DeadlineExceeded, err
	}
	err = cmd.Wait()
	return time.Since(start).Round(time.Millisecond), cctx.Err() == context.DeadlineExceeded, err
}

// streamUsage is the final `result` event's accounting from a stream-json run.
type streamUsage struct {
	InputTokens  int
	OutputTokens int
	NumTurns     int
	CostUSD      float64
	found        bool
	// scanErr is a non-EOF read error (or over-long line) that ended the stream
	// BEFORE the terminating `result` event was seen. The result event carries
	// the ONLY usage numbers and arrives last, so an error mid-stream silently
	// loses token capture; callers surface scanErr instead of mistaking it for a
	// clean text-runtime EOF.
	scanErr error
}

// streamEvent is the subset of a `claude --output-format stream-json` event we
// read: assistant content (for the readable rendering) and the result event's
// usage accounting.
type streamEvent struct {
	Type    string `json:"type"`
	Message struct {
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
	body := fmt.Sprintf("output_tokens: %d\ninput_tokens: %d\nnum_turns: %d\ncost_usd: %.6f\n",
		u.OutputTokens, u.InputTokens, u.NumTurns, u.CostUSD)
	_ = os.WriteFile(filepath.Join(runDir, "usage.txt"), []byte(body), 0o644)
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
	f, _ := clikit.ParseFlags(args)
	// Optional budgets: an agent over either limit is a runaway. --reap kills
	// it (whole tree); without --reap it is only flagged, so you can look first.
	maxRSS := parseBytes(f.Get("max-rss"))           // e.g. 2G, 500M; 0 = no limit
	maxRun := parseDurationArg(f.Get("max-runtime")) // e.g. 15m, 900; 0 = no limit
	reap := f.Bool("reap")
	// --tail: under each agent, print the last non-empty transcript line — its
	// current activity. RAM/CPU alone can't tell a reasoning agent from a wedged
	// one; the live tail can (a thinking agent's last line keeps moving).
	tail := f.Bool("tail")

	live := liveAgents(w)
	for _, rec := range live {
		u := procmon.SampleGroup(rec.PGID)
		age := time.Since(rec.Started).Round(time.Second)
		over := ""
		if maxRSS > 0 && int64(u.RSSKB)*1024 > maxRSS {
			over += fmt.Sprintf(" OVER-RAM(>%s)", humanBytes(maxRSS))
		}
		if maxRun > 0 && age > maxRun {
			over += fmt.Sprintf(" OVER-TIME(>%s)", maxRun)
		}
		// CPUPct is ps's %cpu: cputime/elapsed AVERAGED over each process's whole
		// lifetime, NOT an instantaneous sample. Labelled "CPUavg" so an operator
		// does not read a long-idle agent's high lifetime average as current load.
		fmt.Fprintf(ctx.Stdout, "%s  %-14s %-12s %-10s pid %-7d %2d proc  %8s RAM  %5.0f%% CPUavg  %7s GPU  up %s%s\n",
			rec.RunID[:min(10, len(rec.RunID))], clikit.OrDash(rec.Child), clikit.OrDash(rec.Runtime),
			"task "+clikit.OrDash(rec.Task), rec.PID, u.Procs, humanKB(u.RSSKB), u.CPUPct, gpuStr(u.GPUMiB), age, over)
		if tail {
			line := lastTranscriptLine(filepath.Join(w.RunDir(rec.RunID), "transcript.log"))
			if line == "" {
				line = "(no transcript output yet)"
			}
			fmt.Fprintf(ctx.Stdout, "            ↳ %s\n", truncateLine(line, 100))
		}
		if over != "" && reap {
			killOne(ctx, w, rec, 3*time.Second)
		}
	}
	if len(live) == 0 {
		fmt.Fprintln(ctx.Stdout, "no live agents")
	}
	return nil
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
	f, _ := clikit.ParseFlags(args)
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
	if n, _ := strconv.Atoi(f.Get("tail")); n > 0 {
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
		fh.Close()
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
		if text, _ := renderStreamLine(ln); text != "" {
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
			if text, _ := renderStreamLine(raw); text != "" {
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

// splitClaims parses a comma-separated --claim value into cleaned paths.
func splitClaims(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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

// pidStart captures a freshly-started child's OS start time for its proc.txt,
// so a later reader can tell the real agent from a process that recycled its
// PID. Best-effort: an empty string just falls back to a bare liveness probe.
func pidStart(pid int) string { s, _ := procmon.ProcStart(pid); return s }

// liveAgents reads every run's proc.txt and returns those whose leader process
// is still alive AND still identifies as the spawned agent (PID not recycled),
// newest first.
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
		if procmon.AliveRecord(rec) {
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

// cmdWait blocks until the named detached run(s) finish — or all live agents if
// none are named — then finalizes each one's outcome from the workspace effects
// it left behind. This is the block half of async orchestration: `spawn
// --detach` many, then `wait` on them, instead of hand-rolling shell polling.
func cmdWait(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	interval := 3 * time.Second
	if n, _ := strconv.Atoi(f.Get("interval")); n > 0 {
		interval = time.Duration(n) * time.Second
	}
	overall := 3600
	if n, _ := strconv.Atoi(f.Get("timeout")); n > 0 {
		overall = n
	}

	pending := map[string]procmon.Record{}
	if len(f.Pos) > 0 {
		for _, ref := range f.Pos {
			if rec, ok := readProcByRef(w, ref); ok {
				pending[rec.RunID] = rec
			} else {
				fmt.Fprintf(ctx.Stderr, "no run matching %q\n", ref)
			}
		}
	} else {
		for _, rec := range liveAgents(w) {
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
	for len(pending) > 0 {
		for id, rec := range pending {
			if !procmon.AliveRecord(rec) {
				fmt.Fprintf(ctx.Stdout, "%s  %s (%d of %d)\n", id[:min(10, len(id))], finalizeRun(w, rec), total-len(pending)+1, total)
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
	return nil
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
	runDir := w.RunDir(rec.RunID)
	eventsWS := w
	if raw, e := os.ReadFile(filepath.Join(runDir, "worktree.txt")); e == nil {
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
	// A detached child streamed straight to transcript.log without an in-process
	// parser (the parent had already returned), so usage was never captured live.
	// If the transcript is a stream-json log, harvest its final usage now. Parsing
	// is self-detecting: a plain-text transcript yields no `result` event and
	// nothing is written, so text runtimes are unaffected.
	if _, err := os.Stat(filepath.Join(runDir, "usage.txt")); os.IsNotExist(err) {
		if f, e := os.Open(filepath.Join(runDir, "transcript.log")); e == nil {
			u := teeStreamJSON(f, io.Discard)
			f.Close()
			if u.found {
				writeUsage(runDir, u)
			}
		}
	}
	elapsed := time.Since(rec.Started).Round(time.Second)
	outcome := "done"
	if len(childEvents) == 0 && done == 0 {
		outcome = "no visible result"
	}
	_ = os.WriteFile(filepath.Join(runDir, "outcome.md"),
		[]byte(fmt.Sprintf("outcome: %s (detached)\nchild: %s\nelapsed_since_start: %s\nacceptance: %d/%d\nevents_by_child: %d\n",
			outcome, rec.Child, elapsed, done, total, len(childEvents))), 0o644)
	return fmt.Sprintf("%s: %s · %s · %d event(s) · acceptance %d/%d",
		rec.Child, outcome, elapsed, len(childEvents), done, total)
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
