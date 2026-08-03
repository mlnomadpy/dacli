// Package orchestration is the autonomous-team slice: it runs the whole
// software process as a governed, perpetual loop. A cycle walks the same phases
// a real team walks each sprint — review, plan, implement, test, land, retro —
// and then goes around again, without a human in the loop.
//
// It owns NO agent-spawning or integration logic of its own: every phase is a
// real `dacli` subcommand invocation (spawn, wait, ship, retro), sequenced by
// this driver and gated by a pure Governor. That keeps the slice inside the
// feature-sliced boundary (it imports no sibling feature) and makes every phase
// a first-class, logged run rather than hidden in-process magic.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "loop", Brief: "Run the whole team process as a governed perpetual loop: review→plan→implement→test→land→retro, then repeat (--dry-run to preview, --max-cycles to bound)", Run: cmdLoop},
	{Path: "loop status", Brief: "Show the running/last loop's cycle count, trunk marker, tokens spent this window, and ready backlog size", Run: cmdLoopStatus},
}

// runner executes a dacli subcommand. Real runs shell out to this very binary
// so each phase is a logged, attributable run; tests inject a fake.
type runner interface {
	run(label string, args ...string) (string, error)
}

// execRunner invokes os.Executable() with the given args, inheriting the
// environment (so DACLI_AGENT identity flows into children).
type execRunner struct {
	cwd    string
	stdout *os.File
}

func (r execRunner) run(label string, args ...string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		exe = "dacli"
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = r.cwd
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// loopStack reads the stack `dacli new` recorded on the project the loop is
// about to drive (dacli 192). A missing or stackless project yields the zero
// Stack, which pins every role default to the constant it was before — the
// backwards-compatibility contract for every workspace that predates stacks.
func loopStack(w *workspace.Workspace, project string) prompts.Stack {
	p, err := store.LoadProject(w, project)
	if err != nil {
		return prompts.Stack{}
	}
	return prompts.StackFromProject(p.Doc)
}

// dryRunner logs the intended command and does nothing.
type dryRunner struct{ log func(string) }

func (r dryRunner) run(label string, args ...string) (string, error) {
	r.log(fmt.Sprintf("  would run: dacli %s", strings.Join(args, " ")))
	return "", nil
}

// loopCfg is the resolved policy for one `dacli loop` invocation.
type loopCfg struct {
	project     string
	implRole    string
	reviewRole  string
	width       int   // implementers spawned per cycle
	perCycleTok int64 // --max-tokens passed to each spawn (0 = unset)
	dryRun      bool
	yolo        bool // no between-cycle checkpoint pause
	pr          bool // land through PRs + auto-merge (default true)
}

func cmdLoop(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("project", "impl-role", "review-role", "width", "max-tokens",
		"dry-run", "yolo", "no-pr", "advise", "budget-window", "window-tokens",
		"idle", "max-cycles", "no-progress-halt", "stop-file"); err != nil {
		return err
	}

	project := f.Get("project")
	if project == "" {
		// Default to the sole project if there is exactly one.
		ps, _ := store.ListProjects(w)
		if len(ps) == 1 {
			project = ps[0].Slug
		} else {
			return clikit.Usagef("usage: dacli loop --project <slug> [--width N] [--impl-role R] [--review-role R] [--max-cycles N] [--window-tokens N --budget-window DUR] [--max-tokens N] [--idle DUR] [--no-progress-halt N] [--stop-file PATH] [--no-pr] [--yolo] [--dry-run] [--advise]")
		}
	}

	// Role defaults follow the project's recorded stack (dacli 192). They used
	// to be the constants "fixer" and "go-auditor", so a Python project was
	// audited by a role named for a language it does not use — observed live.
	// An explicit --impl-role/--review-role still wins over everything.
	stack := loopStack(w, project)
	inRoster := func(name string) bool { _, ok := store.LoadRole(w, name); return ok }

	cfg := loopCfg{
		project:     project,
		implRole:    orDefault(f.Get("impl-role"), prompts.RoleFor(stack, "fixer", "fixer", inRoster)),
		reviewRole:  orDefault(f.Get("review-role"), prompts.RoleFor(stack, "auditor", "go-auditor", inRoster)),
		width:       atoiDefault(f.Get("width"), 2),
		perCycleTok: int64(atoiDefault(f.Get("max-tokens"), 0)),
		dryRun:      f.Bool("dry-run"),
		yolo:        f.Bool("yolo"),
		pr:          !f.Bool("no-pr"),
	}

	// --advise (mirrors `spawn --advise`): report the calibrated per-cycle
	// token cost band for this width/role config and return — no agents
	// spawned, no grant needed, the unbounded-loop stop-condition refusal
	// below never even runs.
	if f.Bool("advise") {
		printLoopAdvisory(ctx, w, cfg)
		return nil
	}

	gov := &Governor{
		WindowDur:      parseDurDefault(f.Get("budget-window"), 24*time.Hour),
		WindowTokens:   int64(atoiDefault(f.Get("window-tokens"), 0)),
		Idle:           parseDurDefault(f.Get("idle"), 30*time.Minute),
		MaxCycles:      atoiDefault(f.Get("max-cycles"), 0),
		NoProgressHalt: atoiDefault(f.Get("no-progress-halt"), 3),
		StopFile:       resolveStopFile(w, f.Get("stop-file")),
	}
	// A perpetual loop runs as a fresh process every checkpoint (the default,
	// non-yolo path returns after each cycle for the operator to re-run) — so
	// without this reload every restart would silently forget tokens already
	// spent this window and cycles/thrash-streak already accumulated, and a
	// --window-tokens or --no-progress-halt guard would never actually trip.
	if st, err := readGovernorState(w, project); err == nil {
		gov.Restore(st)
	}

	// A perpetual loop with no bound and no kill switch is a footgun. Require
	// one explicit termination affordance unless the operator opts into --yolo.
	if gov.MaxCycles == 0 && gov.NoProgressHalt == 0 && !cfg.yolo {
		return clikit.Usagef("refusing an unbounded loop with no stop condition: set --max-cycles N, keep --no-progress-halt > 0, or pass --yolo to accept a truly perpetual run (kill it with the stop file: %s)", gov.StopFile)
	}

	var run runner
	if cfg.dryRun {
		run = dryRunner{log: func(s string) { fmt.Fprintln(ctx.Stdout, s) }}
	} else {
		if id.Grant != model.GrantRW {
			return clikit.Refusedf("dacli loop spawns agents and lands PRs — that needs an rw grant (you are %s)", id.Grant)
		}
		run = execRunner{cwd: ctx.Cwd}
	}

	d := &driver{ctx: ctx, w: w, cfg: cfg, gov: gov, run: run, sleep: time.Sleep, now: time.Now}
	return d.loop()
}

// cmdLoopStatus reports the last persisted snapshot of a loop run for a
// project — the running loop's own writes if one is mid-flight, or the final
// snapshot of the last completed run otherwise.
func cmdLoopStatus(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("project"); err != nil {
		return err
	}

	project := f.Get("project")
	if project == "" {
		ps, _ := store.ListProjects(w)
		if len(ps) == 1 {
			project = ps[0].Slug
		} else {
			return clikit.Usagef("usage: dacli loop status --project <slug>")
		}
	}

	st, err := readLoopState(w, project)
	if err != nil {
		return fmt.Errorf("no persisted loop state for project %s — run `dacli loop --project %s` at least once", project, project)
	}

	fmt.Fprintf(ctx.Stdout, "project %s — cycle %d · trunk marker %d · tokens this window %d · ready backlog %d\n",
		st.Project, st.Cycle, st.TrunkMarker, st.WindowTokens, st.Backlog)
	fmt.Fprintf(ctx.Stdout, "last: %s", st.Status)
	if st.Reason != "" {
		fmt.Fprintf(ctx.Stdout, " (%s)", st.Reason)
	}
	if !st.UpdatedAt.IsZero() {
		fmt.Fprintf(ctx.Stdout, " · updated %s", st.UpdatedAt.Format(time.RFC3339))
	}
	fmt.Fprintln(ctx.Stdout)
	return nil
}

// printLoopAdvisory is the body of `loop --advise`: the expected token cost
// of ONE cycle at this width, from measured calibration bands — the P2 loop's
// budgeting sibling to `spawn --advise`'s per-task figure. It changes nothing.
//
// A cycle spends tokens on `width` build spawns (role implRole) plus one
// review spawn (role reviewRole); `wait`/`accept`/`ship`/`retro` run
// in-process and spend none. Bands here group by ROLE ALONE, not the full
// role×model×runtime triple `dacli calibrate` reports — the loop does not pin
// a model or runtime ahead of a spawn, so role is the coarsest grouping this
// projection can honestly commit to (store.TokensPerRun).
func printLoopAdvisory(ctx *clikit.Ctx, w *workspace.Workspace, cfg loopCfg) {
	samples := store.CalibrationSamples(w)
	fmt.Fprintf(ctx.Stdout, "── loop advise · width %d · impl=%s · review=%s ──\n", cfg.width, cfg.implRole, cfg.reviewRole)

	implMed, implP10, implP90, implN := store.TokensPerRun(samples, cfg.implRole)
	reviewMed, reviewP10, reviewP90, reviewN := store.TokensPerRun(samples, cfg.reviewRole)

	report := func(label, role string, med, p10, p90 float64, n int) {
		switch {
		case n >= 10:
			fmt.Fprintf(ctx.Stdout, "  %-6s role %-14s ~%.0f median output-tokens/run  p10–p90 %.0f–%.0f  (n=%d) ← AUTHORITATIVE\n",
				label, role, med, p10, p90, n)
		case n > 0:
			fmt.Fprintf(ctx.Stdout, "  %-6s role %-14s ~%.0f median output-tokens/run  (n=%d, PROVISIONAL — n<10)\n",
				label, role, med, n)
		default:
			fmt.Fprintf(ctx.Stdout, "  %-6s role %-14s no token history yet\n", label, role)
		}
	}
	report("build", cfg.implRole, implMed, implP10, implP90, implN)
	report("review", cfg.reviewRole, reviewMed, reviewP10, reviewP90, reviewN)

	switch {
	case implN > 0 && reviewN > 0:
		expected := float64(cfg.width)*implMed + reviewMed
		low := float64(cfg.width)*implP10 + reviewP10
		high := float64(cfg.width)*implP90 + reviewP90
		conf := "AUTHORITATIVE"
		if implN < 10 || reviewN < 10 {
			conf = "PROVISIONAL — a band above has n<10"
		}
		fmt.Fprintf(ctx.Stdout, "  expected cycle cost at width %d: ~%.0f output tokens  (band %.0f–%.0f)  %s\n",
			cfg.width, expected, low, high, conf)
	case implN > 0 || reviewN > 0:
		fmt.Fprintln(ctx.Stdout, "  expected cycle cost: partial — one role above has no token history yet, so no combined figure")
	default:
		fmt.Fprintln(ctx.Stdout, "  expected cycle cost: no measured band history yet — run some cycles first, then `dacli calibrate`")
	}
	fmt.Fprintln(ctx.Stdout, "── (advice only; no agents spawned) ──")
}

type driver struct {
	ctx             *clikit.Ctx
	w               *workspace.Workspace
	cfg             loopCfg
	gov             *Governor
	run             runner
	sleep           func(time.Duration)
	now             func() time.Time
	trunkBranch     string          // the branch ship/integrate lands into; resolved once
	lastTrunkMarker int             // most recently observed trunkMarker(), for status snapshots
	pendingLand     []string        // self-PR branches opened this run not yet confirmed merged (see recordSelfPR)
	pendingAccept   []pendingAccept // built tasks whose `accept --force` awaits PR-merge confirmation (see reconcilePendingAccepts)
}

// pendingAccept is a self-PR task built this run whose task record is held
// open (not `accept --force`d) until its PR's fate is confirmed. Closing the
// record the moment the PR merely OPENS — the old behavior — marked the task
// done before GitHub's async auto-merge (or a later CI failure) ever
// happened, so the backlog could claim "done" work the trunk never received
// (issue #74 / task 115). Holding it here also excludes the task from the
// ready frontier (see excludePending) so a still-in-flight PR is never
// rebuilt by a subsequent cycle.
type pendingAccept struct {
	Seq    int
	Branch string
}

func (d *driver) logf(format string, a ...any) {
	fmt.Fprintf(d.ctx.Stdout, format+"\n", a...)
}

// saveState persists a status snapshot for `dacli loop status` to read — best
// effort, called at every governor checkpoint.
func (d *driver) saveState(status, reason string, backlog int) {
	writeLoopState(d.w, loopState{
		Project:      d.cfg.project,
		Cycle:        d.gov.Cycle(),
		TrunkMarker:  d.lastTrunkMarker,
		WindowTokens: d.gov.WindowSpent(),
		Backlog:      backlog,
		Status:       status,
		Reason:       reason,
		UpdatedAt:    d.now(),
	})
	writeGovernorState(d.w, d.cfg.project, d.gov.State())
}

func (d *driver) loop() error {
	d.logf("dacli loop — project %s · impl=%s · review=%s · width=%d%s",
		d.cfg.project, d.cfg.implRole, d.cfg.reviewRole, d.cfg.width, dryTag(d.cfg.dryRun))
	if d.gov.MaxCycles > 0 {
		d.logf("bounded to %d cycle(s); stop file: %s", d.gov.MaxCycles, d.gov.StopFile)
	} else {
		d.logf("perpetual; stop file: %s · thrash-halt after %d cycles with no trunk advance", d.gov.StopFile, d.gov.NoProgressHalt)
	}

	d.trunkBranch = d.resolveTrunkBranch()
	prevTrunk := d.trunkMarker()
	d.lastTrunkMarker = prevTrunk

	for {
		// Reconcile the local trunk checkout with origin BEFORE doing anything
		// else this cycle: under --pr --auto, GitHub merges a fixer's PR (and
		// deletes its branch) asynchronously, on its own schedule — not
		// synchronously inside any dacli command this loop runs. Without this,
		// local main only ever falls further behind origin/main across cycles,
		// and the record commit recordSelfPR makes later in this same cycle
		// would sit behind trunk, risking the non-fast-forward push
		// PushSync exists to retry. Best-effort like every other network touch
		// here: it only ever fast-forwards (never discards local work), and a
		// diverged local, missing remote, or wedged network just leaves a note.
		d.syncTrunk()

		// Reconcile every task whose accept is still deferred (built by a prior
		// cycle in --pr mode, awaiting its PR's fate): a confirmed merge closes it
		// now, a PR that closed unmerged drops it from tracking so it re-enters
		// the ready pool for a fresh attempt instead of staying stuck forever —
		// see pendingAccept and reconcilePendingAccepts.
		d.reconcilePendingAccepts()

		// Walk the stage gates as far as they open before choosing this
		// cycle's work — a project sitting in a phase whose gates have all
		// passed is deadlock, not process (see advanceStages, dacli 189).
		d.advanceStages()

		ready, err := readyTasks(d.w, d.cfg.project)
		if err != nil {
			return err
		}
		ready = excludePending(ready, d.pendingAccept)
		rankByPriority(d.w, d.cfg.project, ready)
		dec, why := d.gov.Before(len(ready), d.now())
		d.saveState(dec.String(), why, len(ready))
		switch dec {
		case Halt:
			d.logf("● halt: %s", why)
			return nil
		case SleepWindow:
			rem := d.gov.WindowRemaining(d.now())
			d.logf("● %s (resets in %s)", why, rem.Round(time.Second))
			if d.cfg.dryRun {
				return nil
			}
			d.sleep(rem)
			continue
		case Idle:
			d.logf("● cycle %d: %s", d.gov.Cycle()+1, why)
			// Even with an empty backlog, run a review pass to regenerate work —
			// that is what makes the machine self-feeding rather than stalling.
			// Its spend is charged to the SAME window a runCycle charges — an
			// idle tick is not a sprint (no cycle-counter/thrash-streak bump),
			// but its tokens are real and must still count against
			// --window-tokens, the loop's steady-state cost guard.
			since := store.LatestRunID(d.w)
			d.reviewPhase()
			d.gov.ChargeIdleTokens(store.RunsTokensSince(d.w, since))
			d.saveState(dec.String(), why, len(ready))
			if d.cfg.dryRun {
				return nil
			}
			// An UNPRODUCTIVE idle — review regenerated no ready work — counts
			// toward --max-cycles, so a bounded run on a permanently empty
			// backlog terminates instead of idling forever (dacli 172). A
			// PRODUCTIVE idle (review filed work) is not charged: the build it
			// enables is the cycle the budget is for, and the next iteration
			// proceeds straight to it.
			if after, _ := readyTasks(d.w, d.cfg.project); len(after) == 0 {
				d.gov.CountIdleCycle()
				if d.gov.MaxCyclesReached() {
					d.logf("● reached --max-cycles %d (backlog still empty)", d.gov.MaxCycles)
					return nil
				}
			}
			d.sleep(d.gov.Idle)
			continue
		}

		tokens := d.runCycle(ready)

		// PROGRESS — the thrash guard's signal is REAL trunk advancement, not a
		// task-status delta. Under the default --pr --auto path, merges land on
		// origin ASYNCHRONOUSLY (GitHub merges each PR after its CI passes), so a
		// task that `accept --all` closes this cycle may not have merged yet — or
		// may fail CI and never merge. `landed` is therefore the count of commits
		// that actually reached trunk (local OR origin) since the last cycle. A
		// PR queued this cycle that merges a cycle or two later resets the streak
		// then; only trunk that never moves across NoProgressHalt cycles halts —
		// which is exactly the runaway (PRs that never land) and stall (agents
		// producing nothing) the guard exists to catch.
		curTrunk := d.trunkMarker()
		d.lastTrunkMarker = curTrunk
		landed := curTrunk - prevTrunk
		if landed < 0 {
			landed = 0
		}
		prevTrunk = curTrunk

		dec, why = d.gov.AfterCycle(landed, tokens)
		remaining, _ := readyTasks(d.w, d.cfg.project)
		d.saveState(dec.String(), why, len(remaining))
		if dec == Halt {
			d.logf("● halt: %s", why)
			return nil
		}
		if d.cfg.dryRun {
			d.logf("(dry-run: one cycle previewed; stopping)")
			return nil
		}
		if !d.cfg.yolo {
			d.logf("— cycle %d done (trunk advanced by %d). Checkpoint: re-run to continue, or touch %s to stop —",
				d.gov.Cycle(), landed, d.gov.StopFile)
			return nil
		}
	}
}

// runCycle executes one full sprint: build → test → land → review → retro. It
// returns the tokens charged; trunk-advancement (the thrash-guard signal) is
// measured by the caller across the cycle, not derived from a task-status delta
// here — see loop().
func (d *driver) runCycle(ready []*store.Task) (tokens int64) {
	since := store.LatestRunID(d.w)
	defer func() { tokens = store.RunsTokensSince(d.w, since) }()
	cycle := d.gov.Cycle() + 1
	batch := ready
	if len(batch) > d.cfg.width {
		batch = batch[:d.cfg.width]
	}
	d.logf("● cycle %d — building %d task(s):", cycle, len(batch))

	// BUILD — one detached implementer per task, each opening its own PR. A
	// task only counts as built if BOTH the spawn command itself did not
	// error (a synchronous refusal — taint, budget, malformed flags) AND, once
	// the wave finishes, its dacli/<seq>-slug branch actually exists (catching
	// an async failure: the child crashed or was killed after a clean launch
	// and never committed). A batch task that fails either check must not be
	// force-closed below — the next cycle has to re-pick it, not silently lose it.
	built := make(map[int]bool, len(batch))
	// The role is resolved per cycle, not read straight off the config: on a
	// phase-gated project the configured implementer may have no work in the
	// current phase, and spawning it anyway is a guaranteed refusal (dacli 189).
	// On an untemplated project buildRole returns cfg.implRole unchanged.
	buildRole := d.buildRole()
	for _, t := range batch {
		ref := fmt.Sprintf("%03d", t.Seq)
		spawn := []string{"spawn", "--task", ref, "--role", buildRole, "--detach", "--worktree"}
		if d.cfg.pr {
			spawn = append(spawn, "--pr")
		}
		if d.cfg.perCycleTok > 0 {
			spawn = append(spawn, "--max-tokens", fmt.Sprint(d.cfg.perCycleTok))
		}
		d.logf("  → %s: %s", ref, t.Title)
		if out, err := d.run.run("spawn", spawn...); err != nil {
			d.logf("    spawn refused/failed: %s", firstLine(out))
			continue
		}
		built[t.Seq] = true
	}

	// TEST — block until the detached wave finishes and finalizes.
	d.logf("  waiting on the wave…")
	d.run.run("wait", "wait")

	// Re-check every spawn that launched cleanly: did its branch actually
	// land? A run that started fine can still die mid-flight.
	for _, t := range batch {
		if !built[t.Seq] {
			continue
		}
		branch := taskBranch(t)
		if !d.branchHasWork(branch) {
			d.logf("    %03d: %s has no commits after wait — treating spawn as failed", t.Seq, branch)
			built[t.Seq] = false
		}
	}

	// LAND — two models, chosen by --pr:
	if d.cfg.pr {
		// Self-PR: each fixer opened its own PR and queued GitHub auto-merge
		// (dacli pr --auto), so GitHub lands it on green CI without the loop
		// re-integrating (re-opening a PR on an existing branch would only error).
		// The task record is NOT closed here — accept --force must wait until the
		// PR actually MERGES, or a task marked done here could still fail CI and
		// never land, leaving the backlog claiming work the trunk never received
		// (issue #74 / task 115). Instead every actually-built task is parked in
		// pendingAccept; reconcilePendingAccepts (called at the top of loop())
		// closes it once merged, or drops it (leaving it open for a fresh retry)
		// once its PR closes unmerged. A task whose spawn was refused/failed is
		// left open (not tracked, not box-checked) so the next cycle re-picks it.
		d.logf("  built tasks' accept is deferred until each PR's merge is confirmed…")
		for _, t := range batch {
			if !built[t.Seq] {
				d.logf("    %03d: spawn refused/failed — leaving open for retry", t.Seq)
				continue
			}
			branch := taskBranch(t)
			d.pendingAccept = append(d.pendingAccept, pendingAccept{Seq: t.Seq, Branch: branch})
			d.pendingLand = append(d.pendingLand, branch)
		}
		d.recordSelfPR()
	} else {
		// Local model: fixers committed to their branches without opening PRs, so
		// the loop integrates them into trunk itself.
		d.logf("  integrating done branches…")
		d.run.run("ship", d.shipArgs("--project", d.cfg.project)...)
	}

	// REVIEW — regenerate the backlog: an auditor files the next
	// evidence-based improvement(s) as fresh tasks.
	d.reviewPhase()

	// RETRO — harvest the cycle for the record. cmdRetro requires a ref and at
	// least one bullet; the loop passes the project as the ref and a factual
	// per-cycle bullet, so this records a note instead of exiting 2 unnoticed
	// every cycle (dacli 173).
	builtCount := 0
	for _, t := range batch {
		if built[t.Seq] {
			builtCount++
		}
	}
	d.run.run("retro", "retro", d.cfg.project, "--improve",
		fmt.Sprintf("cycle: %d of %d spawned task(s) produced work; follow-ups are filed as tasks by the review phase", builtCount, len(batch)))

	// The deferred token charge above sums every run this cycle produced
	// (build spawns + the review spawn) from their usage.txt actuals — 0 for
	// any run whose runtime never reported usage, the same honest degrade
	// calibration applies elsewhere.
	return
}

// recordSelfPR commits the .dacli workspace record every cycle but only PUSHES
// it once none of the self-PR branches this loop opened are still awaiting
// GitHub's auto-merge. Pushing while one is in flight advances `main` out from
// under its queued PR, and under strict branch protection (require branches
// up to date) GitHub reads that as "behind" and never merges it — stranding
// every fixer PR, even at --width 1 (task 114 / issue #75). The record commit
// is data-only bookkeeping, so holding its push back never blocks a task from
// closing or a branch from landing; it just keeps `main` stable while a PR is
// pending, and catches the push up the first cycle nothing is left in flight.
func (d *driver) recordSelfPR() {
	d.pendingLand = d.stillPending(d.pendingLand)

	args := d.shipArgs("--no-accept", "--no-integrate", "--project", d.cfg.project)
	if len(d.pendingLand) == 0 {
		args = append(args, "--push")
	} else {
		d.logf("  record: holding the push — %d PR(s) still in flight (%s); pushes once they land", len(d.pendingLand), strings.Join(d.pendingLand, ", "))
	}
	if out, err := d.run.run("record", args...); err != nil {
		d.logf("  record: ship failed: %s", firstLine(out))
	}
}

// stillPending refreshes remote-tracking refs (best-effort — a wedged network
// must never block the loop) and drops any branch GitHub has already merged
// (or closed) and deleted from the returned set. `dacli pr --auto` and
// `dacli integrate --pr` both pass --delete-branch, so a branch's remote-
// tracking ref disappearing after a pruning fetch is the same landed signal
// trunkMarker/branchExists already lean on elsewhere in this file.
func (d *driver) stillPending(branches []string) []string {
	if len(branches) == 0 {
		return branches
	}
	if !d.cfg.dryRun {
		gitx.RunNetwork(d.w.Root, "fetch", "-q", "--prune", "origin")
	}
	still := branches[:0]
	for _, b := range branches {
		if _, err := d.git("rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+b); err == nil {
			still = append(still, b)
		}
	}
	return still
}

// reconcilePendingAccepts checks every task whose accept is still deferred
// (built by a --pr cycle, parked in pendingAccept) against its PR's real
// fate: a confirmed merge closes the task record now — the only point at
// which the backlog is allowed to claim it done, per task 115/issue #74. A PR
// that closed without merging drops the task from tracking so the
// (still-open) task re-enters the ready pool for a fresh attempt instead of
// being stuck behind a rejected PR forever. Anything still open or
// unanswerable is left pending for the next check.
func (d *driver) reconcilePendingAccepts() {
	if len(d.pendingAccept) == 0 {
		return
	}
	remaining := d.pendingAccept[:0]
	for _, p := range d.pendingAccept {
		switch d.prLandStatus(p.Branch) {
		case "merged":
			d.logf("    %03d: PR merged — closing the task record", p.Seq)
			d.run.run("accept", "accept", fmt.Sprintf("%03d", p.Seq), "--force")
			d.gcBranch(p.Branch)
		case "orphaned":
			d.logf("    %03d: PR closed without merging — leaving open for a fresh retry", p.Seq)
		default: // "landing" (PR still open) or "unknown" (gh/network unreachable)
			remaining = append(remaining, p)
		}
	}
	d.pendingAccept = remaining
}

// gcBranch removes a task's worktree and local branch once its work has landed
// on trunk (a confirmed PR merge). Without this the workspace accumulates a
// worktree and a stale branch per completed task indefinitely — a live
// workspace reached 72 worktrees / 71 merged local branches (dacli 182). Only
// called on a CONFIRMED merge, never a blanket sweep: a zero-commit branch of a
// still-running agent is trivially an ancestor of trunk, so sweeping by
// ancestry would risk deleting a live worktree mid-work. Best-effort — a GC
// failure never blocks the loop.
func (d *driver) gcBranch(branch string) {
	if wts, err := gitx.ListWorktrees(d.w.Root); err == nil {
		for _, wt := range wts {
			if wt.Branch == branch {
				gitx.RemoveWorktree(d.w.Root, wt.Path)
				break
			}
		}
	}
	d.git("branch", "-D", branch)
}

// runGH runs the GitHub CLI for the loop's own merge-confirmation checks. A
// package variable so a test can stub it, mirroring features/vcs's identical
// runGH — duplicated rather than imported because the feature-slice isolation
// rule (arch_test's TestFeatureSlicesAreIsolated) forbids orchestration
// importing vcs (see taskBranch above for the same reasoning).
var runGH = func(dir string, args ...string) (string, error) {
	pctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	c := exec.CommandContext(pctx, "gh", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// prLandStatus classifies whether branch has actually reached trunk:
//   - "merged"   — the branch's work is on trunk now.
//   - "landing"  — a PR is open; GitHub may merge it any moment.
//   - "orphaned" — no open PR and the branch never merged: really stuck.
//   - "unknown"  — gh and a trunk fetch both failed to answer.
//
// This mirrors features/vcs's checkLanded (gh state first, a fresh-fetch
// ancestor check only when gh finds no PR) but is duplicated, not imported —
// same feature-slice isolation reasoning as runGH above.
func (d *driver) prLandStatus(branch string) string {
	if out, err := runGH(d.w.Root, "pr", "list", "--head", branch, "--state", "all", "--json", "state", "--limit", "1"); err == nil {
		var prs []struct {
			State string `json:"state"`
		}
		if jerr := json.Unmarshal([]byte(out), &prs); jerr == nil && len(prs) > 0 {
			switch strings.ToUpper(prs[0].State) {
			case "MERGED":
				return "merged"
			case "OPEN":
				return "landing"
			case "CLOSED":
				return "orphaned"
			}
		}
	}
	if d.cfg.dryRun {
		return "unknown"
	}
	b := d.trunkBranch
	if b == "" {
		b = "main"
	}
	if _, err := gitx.RunNetwork(d.w.Root, "fetch", "-q", "origin", "--", b); err != nil {
		return "unknown"
	}
	// A branch with no commits beyond trunk is trivially an ancestor of it, but
	// it carries no work — a spawn that died before committing. Never report
	// that as "merged" (dacli 168): that is exactly the path that force-accepts
	// an empty branch as a done task.
	if n, err := d.git("rev-list", "--count", "origin/"+b+".."+branch); err == nil && strings.TrimSpace(n) == "0" {
		return "orphaned"
	}
	ok, err := gitx.IsAncestor(d.w.Root, branch, "origin/"+b)
	if err != nil {
		return "unknown"
	}
	if ok {
		return "merged"
	}
	return "orphaned"
}

// excludePending drops every task whose Seq is parked in pending from the
// ready frontier — a task built this loop's --pr cycle stays open (never
// accepted) until its PR's merge is confirmed, so without this it would be
// picked up and rebuilt by the very next cycle while its first PR is still
// in flight.
func excludePending(tasks []*store.Task, pending []pendingAccept) []*store.Task {
	if len(pending) == 0 {
		return tasks
	}
	skip := make(map[int]bool, len(pending))
	for _, p := range pending {
		skip[p.Seq] = true
	}
	out := tasks[:0]
	for _, t := range tasks {
		if !skip[t.Seq] {
			out = append(out, t)
		}
	}
	return out
}

// resolveTrunkBranch finds the branch ship/integrate lands into — the repo's
// default branch — so trunk advancement is measured against the right ref.
func (d *driver) resolveTrunkBranch() string {
	if out, err := d.git("rev-parse", "--abbrev-ref", "origin/HEAD"); err == nil {
		s := strings.TrimSpace(out) // "origin/main"
		if i := strings.LastIndex(s, "/"); i >= 0 {
			s = s[i+1:]
		}
		if s != "" && s != "HEAD" {
			return s
		}
	}
	for _, b := range []string{"main", "master"} {
		if _, err := d.git("rev-parse", "--verify", "--quiet", b); err == nil {
			return b
		}
	}
	if out, err := d.git("rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		if s := strings.TrimSpace(out); s != "" {
			return s
		}
	}
	return "main"
}

// trunkMarker is a monotonic count of commits that have reached trunk — local
// OR origin — so it captures both in-cycle local integrations and the async
// GitHub auto-merges the default --pr --auto path produces. Best-effort: it
// refreshes the remote-tracking ref first (so async auto-merges become visible)
// and degrades to the local count, then 0, when there is no remote or git is
// unavailable.
func (d *driver) trunkMarker() int {
	b := d.trunkBranch
	if b == "" {
		b = "main"
	}
	if !d.cfg.dryRun {
		// Network-bound: a hung fetch (wedged network, a credential prompt) must
		// not block the loop — it gets the longer network leash and, on timeout,
		// this degrades to the local-only rev-list count below, the existing
		// best-effort fallback.
		gitx.RunNetwork(d.w.Root, "fetch", "-q", "origin", "--", b)
	}
	for _, refs := range [][]string{{b, "origin/" + b}, {b}, {"origin/" + b}} {
		args := append([]string{"rev-list", "--count"}, refs...)
		// Exclude the loop's OWN bookkeeping: recordSelfPR commits a .dacli-only
		// record onto trunk every cycle, so counting all commits would make
		// `landed` >= 1 unconditionally and the thrash guard (NoProgressHalt)
		// could never fire. Progress is CODE reaching trunk, not the loop
		// narrating itself (dacli 171).
		args = append(args, "--", ":(exclude).dacli")
		if out, err := d.git(args...); err == nil {
			var n int
			if _, e := fmt.Sscanf(strings.TrimSpace(out), "%d", &n); e == nil {
				return n
			}
		}
	}
	return 0
}

// syncTrunk fast-forwards the local trunk checkout up to origin's latest —
// the reconciliation step for a `gh pr merge --auto` landing that happened
// between cycles. It only ever fast-forwards: gitx.FastForward refuses (and
// this just logs, never fails the loop) the moment local has a commit origin
// lacks, e.g. a record commit made but not yet pushed in a prior cycle — that
// case is left for recordSelfPR's own push (via gitx.PushSync) to reconcile
// with a rebase.
func (d *driver) syncTrunk() {
	if d.cfg.dryRun {
		return
	}
	b := d.trunkBranch
	if b == "" {
		b = "main"
	}
	if out, err := gitx.FastForward(d.w.Root, b); err != nil {
		d.logf("  note: local %s not fast-forwarded to origin: %s", b, firstLine(out))
	}
}

// git runs a local (non-network) git op under gitx's short deadline, so a
// wedged git child (an index lock, a credential-helper prompt) can never
// block the loop indefinitely.
func (d *driver) git(args ...string) (string, error) {
	return gitx.Run(d.w.Root, args...)
}

// shipArgs prepends "ship" and appends --into <trunk> to a ship invocation.
// ship defaults --into to "main" and refuses up front when the checkout is not
// that branch, so on a repo whose trunk is master/renamed the loop's LAND and
// record-ship steps would fail every cycle without forwarding the resolved
// trunk (dacli 174). Omitted when the trunk could not be resolved, letting ship
// keep its own default.
func (d *driver) shipArgs(rest ...string) []string {
	args := append([]string{"ship"}, rest...)
	if d.trunkBranch != "" {
		args = append(args, "--into", d.trunkBranch)
	}
	return args
}

// taskBranch is the task-branch naming convention, duplicated (not imported)
// from features/vcs.BranchFor: the feature-slice isolation rule (arch_test's
// TestFeatureSlicesAreIsolated) forbids orchestration importing vcs, and this
// is the one fact of that convention the loop needs to verify a spawn actually
// produced a branch.
func taskBranch(t *store.Task) string {
	return fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug)
}

// branchExists reports whether branch exists either as a local ref or as an
// already-fetched remote-tracking ref — a worktree spawn commits locally, a
// --pr spawn additionally pushes, and trunkMarker's fetch may or may not have
// run yet, so both are checked.
func (d *driver) branchExists(branch string) bool {
	if _, err := d.git("rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		return true
	}
	if _, err := d.git("rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+branch); err == nil {
		return true
	}
	return false
}

// branchHasWork reports whether branch carries at least one commit beyond the
// trunk it was forked from. The worktree+branch is created at SPAWN time
// (gitx.AddWorktree), so the branch exists at trunk's tip before the child has
// done anything — existence alone is NOT evidence of work. A child that OOMs,
// is killed, or is refused by its runtime right after launch leaves a
// zero-commit branch, which is trivially an ancestor of trunk and would
// otherwise be misread as "merged" and force-accepted as done with no work
// (dacli 168). Compares against a ref that exists (local trunk, else
// origin/trunk); on any git error it returns true, so an unmeasurable branch
// is never destroyed on a false negative — the PR/ancestor checks still apply.
func (d *driver) branchHasWork(branch string) bool {
	if !d.branchExists(branch) {
		return false
	}
	base := d.trunkBranch
	if base == "" {
		base = "main"
	}
	for _, ref := range []string{base, "origin/" + base} {
		if out, err := d.git("rev-list", "--count", ref+".."+branch); err == nil {
			return strings.TrimSpace(out) != "0"
		}
	}
	return true
}

// maxStageAdvancesPerCycle bounds advanceStages. A manifest is a handful of
// stages, and a project whose gates ALL open at once should reach its
// implementation phase in one cycle rather than crawling one stage per cycle;
// the cap only guards against a pathological manifest looping forever.
const maxStageAdvancesPerCycle = 8

// templateStage reads the project's current stage straight off its
// frontmatter. Deliberately not gates.Status: Status evaluates every predicate
// for the stage, and the `command:`/`coverage:` predicates shell out (each with
// a ten-minute leash). This is the cheap "is this project gated at all?"
// question, and for the overwhelmingly common untemplated/solo project it is
// the ONLY gates-related work the loop does. Returns "" when the project has no
// template, and "complete" once every gate has been passed.
func (d *driver) templateStage() string {
	p, err := store.LoadProject(d.w, d.cfg.project)
	if err != nil {
		return ""
	}
	s, _ := p.Doc.Front.Get("template_stage")
	return s
}

// advanceStages walks the project's stage gates as far as they will open, once
// per cycle.
//
// The loop used to be stage-BLIND — this package imported `gates` nowhere — so
// a project on a phase-gated template deadlocked on cycle one. `dacli init
// --template product` starts a project in the DISCOVERY phase, which admits
// only researcher and reviewer kinds, and spawn's phaseGate refuses an
// implementer there ("advance the stage first"). Nothing in an autonomous run
// ever advanced it: `dacli stage advance` is a command a human types. So every
// build spawn was refused, forever, until the thrash guard killed the run
// (dacli 189).
//
// The gate's purpose is to stop work moving on before its preconditions hold.
// Once every check for the current stage passes, that purpose is served and
// keeping the project there is pure deadlock — so the loop advances it itself
// and says so in the log, exactly as `stage advance` would. A closed gate is
// still an answer: the loop reports what is unmet and works the current phase.
//
// Untemplated (solo) projects have no stages and return on the first line, so
// their behavior is identical to before. Dry runs report and mutate nothing.
func (d *driver) advanceStages() {
	for i := 0; i < maxStageAdvancesPerCycle; i++ {
		stage := d.templateStage()
		if stage == "" || stage == "complete" {
			return // untemplated (solo), or already through every gate
		}
		if d.cfg.dryRun {
			// Status is read-only; Advance rewrites the project file.
			st, err := gates.Status(d.w, d.cfg.project)
			if err != nil || st.Complete {
				return
			}
			if unmet := unmetChecks(st.Checks); len(unmet) > 0 {
				d.logf("  stage: %s holds at %q — %d gate check(s) unmet (first: %s)", d.cfg.project, stage, len(unmet), unmet[0])
			} else {
				d.logf("  stage: every gate check at %q passes — would advance", stage)
			}
			return
		}
		newStage, unmet, err := gates.Advance(d.w, d.cfg.project)
		if err != nil {
			d.logf("  stage: gates unreadable (%v) — leaving %s at %q", err, d.cfg.project, stage)
			return
		}
		if len(unmet) > 0 {
			why := unmet[0].Desc
			if unmet[0].Why != "" {
				why += " — " + unmet[0].Why
			}
			d.logf("  stage: %s holds at %q — %d gate check(s) unmet (first: %s)", d.cfg.project, stage, len(unmet), why)
			return
		}
		d.logf("  stage: every gate check at %q passed — advanced %s to %q", stage, d.cfg.project, newStage)
	}
}

// unmetChecks returns the descriptions of the failing checks, for the log.
func unmetChecks(checks []gates.Check) []string {
	var out []string
	for _, c := range checks {
		if !c.OK {
			out = append(out, c.Desc)
		}
	}
	return out
}

// buildRole resolves the role this cycle's BUILD phase spawns with. On an
// untemplated project — the common solo case — it is cfg.implRole verbatim and
// nothing else here runs, so the ungated loop is unchanged.
//
// On a phase-gated project the phase decides which role KINDS may act, and
// spawning a kind the phase does not admit is a guaranteed refusal, cycle after
// cycle (dacli 189). So the loop asks the same question spawn's phaseGate asks,
// and when the configured implementer has no work in this phase it looks for a
// roster role that does. A role with no declared kind is exempt from phase
// gating (phaseGate's own rule), and so is one this workspace does not define —
// both pass through untouched, and any refusal they earn is the same refusal
// they would have earned before.
func (d *driver) buildRole() string {
	ph, err := gates.PhaseFor(d.w, d.cfg.project)
	if err != nil || !ph.Gated {
		return d.cfg.implRole
	}
	role, ok := store.LoadRole(d.w, d.cfg.implRole)
	if !ok || role.Kind == "" || ph.AllowsKind(role.Kind) {
		return d.cfg.implRole
	}
	roles, err := store.LoadRoles(d.w)
	if err != nil {
		return d.cfg.implRole
	}
	if pick := pickRoleForPhase(roles, ph); pick != "" {
		d.logf("  phase %s has no work for %s (kind %s) — building with %s instead",
			ph.Name, d.cfg.implRole, role.Kind, pick)
		return pick
	}
	// Nothing on the roster fits. Falling through to the configured role keeps
	// the pre-existing behavior (a logged refusal) rather than inventing a
	// role; the log says why, because "spawn refused" alone does not.
	d.logf("  phase %s admits only %s and no role on the roster is one — the build spawn will be refused; add a role or advance the stage",
		ph.Name, strings.Join(ph.Allows, ", "))
	return d.cfg.implRole
}

// pickRoleForPhase chooses a roster role whose kind the phase admits, honoring
// the manifest's own `allow:` ORDER as the preference: a template that writes
// "allow: implementer, reviewer" is stating that the implementer is the one who
// should be doing the work there, with the reviewer as the fallback. Ties
// within a kind break by name, so the same roster always yields the same
// choice — a loop that picked a different builder each cycle would be
// unauditable.
func pickRoleForPhase(roles []team.Role, ph gates.Phase) string {
	for _, kind := range ph.Allows {
		var names []string
		for _, r := range roles {
			if r.Name != "" && r.Kind == kind {
				names = append(names, r.Name)
			}
		}
		if len(names) > 0 {
			sort.Strings(names)
			return names[0]
		}
	}
	return ""
}

// reviewPhase spawns a reviewer against the project's standing
// continuous-improvement task, whose charter is to file the single
// highest-value, evidence-based change as new work — never to implement it.
func (d *driver) reviewPhase() {
	ref, err := d.ensureImproveTask()
	if err != nil {
		d.logf("  review: could not seed the improvement task: %v", err)
		return
	}
	d.logf("  review: %s audits and files the next improvement…", d.cfg.reviewRole)
	spawn := []string{"spawn", "--task", ref, "--role", d.cfg.reviewRole}
	if d.cfg.perCycleTok > 0 {
		spawn = append(spawn, "--max-tokens", fmt.Sprint(d.cfg.perCycleTok))
	}
	d.run.run("review", spawn...)
}

// The two review-phase anchor charters. Both carry
// store.ContinuousImprovementMarker so IsLoopAnchor keeps excluding them from
// the ready frontier — an anchor is a standing prompt, never implementer work —
// and they differ only in the suffix, which is how ensureImproveTask tells them
// apart and reuses the right one across cycles.
const (
	evidenceAnchorTitle = store.ContinuousImprovementMarker + ": file the single highest-value evidence-based change"
	specAnchorTitle     = store.ContinuousImprovementMarker + ": decompose the goal into a dependency-ordered backlog"
)

// ensureImproveTask returns the ref of the standing improvement task for the
// project, creating it (open) if absent. The task is the review phase's anchor:
// an auditor is spawned against it every cycle and files fresh work. Which
// anchor is right depends on whether the project has anything to reason FROM —
// see anchorCharter.
func (d *driver) ensureImproveTask() (string, error) {
	title, context, accept := d.anchorCharter()
	for _, st := range []model.Status{model.StatusOpen, model.StatusActive} {
		ts, _ := store.ListTasks(d.w, d.cfg.project, st)
		for _, t := range ts {
			if t.IsLoopAnchor() && t.Title == title {
				return fmt.Sprintf("%03d", t.Seq), nil
			}
		}
	}
	if d.cfg.dryRun {
		return "IMPROVE", nil // placeholder ref for the preview
	}
	t, err := store.CreateTask(d.w, "loop", d.cfg.project, title, store.TaskOpts{
		Priority: "should",
		Context:  context,
		Accept:   accept,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%03d", t.Seq), nil
}

// anchorCharter picks which standing anchor this cycle's review phase runs
// against.
//
// The evidence-based anchor is the original and stays the default: find the ONE
// highest-value improvement grounded in a failing test, a reviewer finding, or a
// real defect, and do NOT invent speculative work. That charter is exactly right
// for a live codebase — and a dead end for a greenfield repo. With no code and
// no backlog there is no evidence to point at, so the auditor correctly files
// nothing, and the loop prints "backlog empty — no evidence-based work; idling
// rather than inventing work" forever. A project with a Goal and zero tasks
// never starts (dacli 190).
//
// So a project that has not been decomposed yet gets the opposite charter: turn
// the stated intent — Goal, Constraints, Out of scope, Success criteria — into a
// dependency-ordered backlog. That anchor IS licensed to invent the work,
// because writing down what a stated goal requires is not speculation; it is the
// planning step the evidence-based anchor presupposes somebody already did.
func (d *driver) anchorCharter() (title, context string, accept []string) {
	if d.preImplementation() {
		return specAnchorTitle,
			fmt.Sprintf("Standing anchor for the autonomous review phase on a project that has NO backlog and NO code yet. There is nothing to survey, so — unlike the evidence-based anchor — you ARE licensed to invent the work here, from the project's own stated intent and from nothing else. Read `dacli project show %s` and decompose its Goal, Constraints, Out of scope, and Success criteria into the smallest set of concrete, buildable tasks that would actually satisfy them. File each with `dacli task add <title> --project %s --accept <criterion>` and REAL acceptance criteria — a task whose acceptance is \"it works\" is as empty as TBD. Order them: `--depends-on <ref>` for work that genuinely cannot start until another task is done, `--parent <ref>` for a step that is part of a larger one. Every task must trace to a line of the Goal or the Success criteria; anything you cannot trace is scope you invented, and belongs in Out of scope instead. Do NOT implement anything here.", d.cfg.project, d.cfg.project),
			[]string{
				"Filed a dependency-ordered set of tasks derived from the project's Goal and Success criteria",
				"Every filed task carries concrete acceptance criteria",
				"Did not implement any change in this task",
			}
	}
	return evidenceAnchorTitle,
		fmt.Sprintf("Standing anchor for the autonomous review phase. Survey the code, tests, CI, and open findings; identify the ONE highest-value improvement grounded in evidence (a failing test, a reviewer finding, a real defect). Before filing, run `dacli task list --project %s --status open` (and --status active) to check whether the backlog already queues it — a prior cycle may have filed the same issue under different wording. `dacli task add` refuses (exit 3) a title that scores as a near-duplicate of an existing open task, so pick real, distinct scope rather than re-filing and re-running with --force. File it with concrete acceptance criteria. Do NOT implement it here, and do NOT invent speculative work.", d.cfg.project),
		[]string{
			"Filed at least one new task grounded in an observed defect, finding, or failing check",
			"Did not implement any change in this task",
		}
}

// preImplementation reports whether the project has nothing an evidence-based
// reviewer could stand on. Both halves matter:
//
//   - No filed work at all. The anchors themselves do not count — they are
//     standing prompts, not backlog. One real task means somebody already
//     decomposed the goal, and re-decomposing it would duplicate their work.
//   - No source in the repository. An empty backlog over a REAL codebase still
//     has evidence to survey (tests, defects, findings), and that project keeps
//     the original charter — this must not change what a working repo does.
func (d *driver) preImplementation() bool {
	tasks, err := store.ListTasks(d.w, d.cfg.project, "")
	if err != nil {
		return false // an unreadable backlog is not evidence of an empty one
	}
	for _, t := range tasks {
		if !t.IsLoopAnchor() {
			return false
		}
	}
	return !d.repoHasCode()
}

// repoHasCode reports whether the repository carries anything an evidence-based
// reviewer could actually audit. Tracked files only: an untracked scratch file
// is not the project. The workspace record (.dacli) is excluded because it is
// the loop's own bookkeeping, and prose is excluded because a repo holding a
// README and a licence has a description of software, not software. On any git
// error this answers TRUE — an unmeasurable repo keeps the pre-existing
// evidence-based anchor rather than being handed a licence to invent work.
func (d *driver) repoHasCode() bool {
	out, err := d.git("ls-files")
	if err != nil {
		return true
	}
	for _, f := range strings.Split(out, "\n") {
		f = strings.TrimSpace(f)
		if f == "" || strings.HasPrefix(f, workspace.Dir+"/") {
			continue
		}
		switch strings.ToLower(filepath.Ext(f)) {
		case ".md", ".txt", "":
			continue
		}
		return true
	}
	return false
}

// readyTasks returns open tasks whose blocking (finish-relation) dependencies
// are all done — the workable frontier the loop draws from.
func readyTasks(w *workspace.Workspace, project string) ([]*store.Task, error) {
	open, err := store.ListTasks(w, project, model.StatusOpen)
	if err != nil {
		return nil, err
	}
	if len(open) == 0 {
		return nil, nil
	}
	done, _ := store.ListTasks(w, project, model.StatusDone)
	isDone := map[string]bool{}
	for _, t := range done {
		isDone[fmt.Sprintf("%03d", t.Seq)] = true
		isDone[t.Slug] = true
	}
	var ready []*store.Task
	for _, t := range open {
		// The standing improvement task is the review phase's anchor, not
		// implementer work — never hand it to a builder.
		if t.IsLoopAnchor() {
			continue
		}
		// A `wont` task is a recorded out-of-scope decision; the loop must not
		// spend an implementer on it (dacli 199).
		if !model.Priority(t.Priority()).Schedulable() {
			continue
		}
		blocked := false
		for _, dep := range t.Deps() {
			if dep.Type == "SS" || dep.Type == "SF" {
				continue // start-relations don't block *starting* this task
			}
			if !isDone[dep.Ref] {
				blocked = true
				break
			}
		}
		if !blocked {
			ready = append(ready, t)
		}
	}
	return ready, nil
}

// rankByPriority orders the ready frontier by MoSCoW priority rank, then
// critical-path slack when a CPM schedule can be computed, then Seq as the
// final tiebreak — mirroring cmdNext's selection (insight.go cmdNext) so the
// loop's BUILD phase and `dacli next` agree on what to work on first. Without
// this, a low-seq could/should would be built ahead of a higher-seq must and
// the critical path would be ignored, contradicting the loop's own
// MoSCoW/critical-path-first charter. Sorts in place.
func rankByPriority(w *workspace.Workspace, project string, ready []*store.Task) {
	if len(ready) < 2 {
		return
	}
	slack, haveCPM := criticalPathSlack(w, project)
	sort.SliceStable(ready, func(i, j int) bool {
		pi, pj := model.Priority(ready[i].Priority()).Rank(), model.Priority(ready[j].Priority()).Rank()
		if pi != pj {
			return pi < pj
		}
		if haveCPM && slack[ready[i].ID] != slack[ready[j].ID] {
			return slack[ready[i].ID] < slack[ready[j].ID]
		}
		return ready[i].Seq < ready[j].Seq
	})
}

// criticalPathSlack computes CPM slack for every open (non-done, non-blocked)
// task in the project. Duplicated from insight.cmdNext's CPM block rather
// than imported — the feature-slice isolation rule (TestFeatureSlicesAreIsolated)
// forbids orchestration importing a sibling feature. Degrades to
// haveCPM=false when any open task is missing an estimate, same as cmdNext.
func criticalPathSlack(w *workspace.Workspace, project string) (map[string]float64, bool) {
	tasks, err := store.ListTasks(w, project, "")
	if err != nil {
		return nil, false
	}
	byRef := map[string]*store.Task{}
	openIDs := map[string]bool{}
	var open []*store.Task
	for _, t := range tasks {
		for _, ref := range []string{t.ID, strings.TrimPrefix(t.ID, "t-"), t.Slug, fmt.Sprintf("%03d", t.Seq)} {
			byRef[ref] = t
		}
		if t.Status != model.StatusDone && t.Status != model.StatusBlocked {
			open = append(open, t)
			openIDs[t.ID] = true
		}
	}

	var nodes []spm.Node
	var edges []spm.Edge
	for _, t := range open {
		est, ok := t.Estimate()
		if !ok {
			return nil, false
		}
		nodes = append(nodes, spm.Node{ID: t.ID, Duration: est.Expected()})
		for _, d := range t.Deps() {
			if dep, ok := byRef[d.Ref]; ok && openIDs[dep.ID] {
				edges = append(edges, spm.Edge{From: dep.ID, To: t.ID, Type: spm.DepType(d.Type)})
			}
		}
	}
	net, err := spm.ComputeCPM(nodes, edges)
	if err != nil {
		return nil, false
	}
	slack := map[string]float64{}
	for id, s := range net.Schedules {
		slack[id] = s.Slack
	}
	return slack, true
}

// --- small helpers ---

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func atoiDefault(v string, def int) int {
	if v == "" {
		return def
	}
	n := def
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return def
	}
	return n
}

func parseDurDefault(v string, def time.Duration) time.Duration {
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	return def
}

func resolveStopFile(w *workspace.Workspace, v string) string {
	if v == "" {
		return filepath.Join(w.Root, ".dacli", "STOP")
	}
	if filepath.IsAbs(v) {
		return v
	}
	return filepath.Join(w.Root, v)
}

func dryTag(dry bool) string {
	if dry {
		return " · DRY-RUN"
	}
	return ""
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
