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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
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

	cfg := loopCfg{
		project:     project,
		implRole:    orDefault(f.Get("impl-role"), "fixer"),
		reviewRole:  orDefault(f.Get("review-role"), "go-auditor"),
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
	trunkBranch     string // the branch ship/integrate lands into; resolved once
	lastTrunkMarker int    // most recently observed trunkMarker(), for status snapshots
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
		ready, err := readyTasks(d.w, d.cfg.project)
		if err != nil {
			return err
		}
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
	for _, t := range batch {
		ref := fmt.Sprintf("%03d", t.Seq)
		spawn := []string{"spawn", "--task", ref, "--role", d.cfg.implRole, "--detach", "--worktree"}
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
		if !d.branchExists(branch) {
			d.logf("    %03d: no %s branch after wait — treating spawn as failed", t.Seq, branch)
			built[t.Seq] = false
		}
	}

	// LAND — two models, chosen by --pr:
	if d.cfg.pr {
		// Self-PR: each fixer opened its own PR and queued GitHub auto-merge
		// (dacli pr --auto), so GitHub lands it on green CI without the loop
		// re-integrating (re-opening a PR on an existing branch would only error).
		// The loop closes every ACTUALLY BUILT task's record here — otherwise the
		// next cycle re-picks a still-open task and reworks it — then commits the
		// workspace state. Whether a PR ACTUALLY merged is tracked separately by
		// trunk advancement in loop(), so closing the record never inflates the
		// thrash-guard's progress signal. A task whose spawn was refused/failed is
		// left open (not closed, not box-checked) so the next cycle re-picks it.
		d.logf("  closing built tasks; their PRs auto-merge on green CI…")
		for _, t := range batch {
			if !built[t.Seq] {
				d.logf("    %03d: spawn refused/failed — leaving open for retry", t.Seq)
				continue
			}
			d.run.run("accept", "accept", fmt.Sprintf("%03d", t.Seq), "--force")
		}
		d.run.run("record", "ship", "--no-accept", "--no-integrate", "--push", "--project", d.cfg.project)
	} else {
		// Local model: fixers committed to their branches without opening PRs, so
		// the loop integrates them into trunk itself.
		d.logf("  integrating done branches…")
		d.run.run("ship", "ship", "--project", d.cfg.project)
	}

	// REVIEW — regenerate the backlog: an auditor files the next
	// evidence-based improvement(s) as fresh tasks.
	d.reviewPhase()

	// RETRO — harvest the cycle for the record.
	d.run.run("retro", "retro", "--project", d.cfg.project)

	// The deferred token charge above sums every run this cycle produced
	// (build spawns + the review spawn) from their usage.txt actuals — 0 for
	// any run whose runtime never reported usage, the same honest degrade
	// calibration applies elsewhere.
	return
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
		gitx.RunNetwork(d.w.Root, "fetch", "-q", "origin", b)
	}
	for _, refs := range [][]string{{b, "origin/" + b}, {b}, {"origin/" + b}} {
		args := append([]string{"rev-list", "--count"}, refs...)
		if out, err := d.git(args...); err == nil {
			var n int
			if _, e := fmt.Sscanf(strings.TrimSpace(out), "%d", &n); e == nil {
				return n
			}
		}
	}
	return 0
}

// git runs a local (non-network) git op under gitx's short deadline, so a
// wedged git child (an index lock, a credential-helper prompt) can never
// block the loop indefinitely.
func (d *driver) git(args ...string) (string, error) {
	return gitx.Run(d.w.Root, args...)
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

// ensureImproveTask returns the ref of the standing improvement task for the
// project, creating it (open) if absent. The task is the review phase's anchor:
// an auditor is spawned against it every cycle and files fresh work.
func (d *driver) ensureImproveTask() (string, error) {
	const marker = "Continuous improvement"
	for _, st := range []model.Status{model.StatusOpen, model.StatusActive} {
		ts, _ := store.ListTasks(d.w, d.cfg.project, st)
		for _, t := range ts {
			if strings.HasPrefix(t.Title, marker) {
				return fmt.Sprintf("%03d", t.Seq), nil
			}
		}
	}
	if d.cfg.dryRun {
		return "IMPROVE", nil // placeholder ref for the preview
	}
	t, err := store.CreateTask(d.w, "loop", d.cfg.project, marker+": file the single highest-value evidence-based change", store.TaskOpts{
		Priority: "should",
		Context:  fmt.Sprintf("Standing anchor for the autonomous review phase. Survey the code, tests, CI, and open findings; identify the ONE highest-value improvement grounded in evidence (a failing test, a reviewer finding, a real defect). Before filing, run `dacli task list --project %s --status open` (and --status active) to check whether the backlog already queues it — a prior cycle may have filed the same issue under different wording. `dacli task add` refuses (exit 3) a title that scores as a near-duplicate of an existing open task, so pick real, distinct scope rather than re-filing and re-running with --force. File it with concrete acceptance criteria. Do NOT implement it here, and do NOT invent speculative work.", d.cfg.project),
		Accept:   []string{"Filed at least one new task grounded in an observed defect, finding, or failing check", "Did not implement any change in this task"},
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%03d", t.Seq), nil
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
		if strings.HasPrefix(t.Title, "Continuous improvement") {
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
