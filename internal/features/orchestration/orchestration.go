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
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "loop", Brief: "Run the whole team process as a governed perpetual loop: review→plan→implement→test→land→retro, then repeat (--dry-run to preview, --max-cycles to bound)", Run: cmdLoop},
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
			return clikit.Usagef("usage: dacli loop --project <slug> [--width N] [--impl-role R] [--review-role R] [--max-cycles N] [--window-tokens N --budget-window DUR] [--max-tokens N] [--idle DUR] [--no-progress-halt N] [--stop-file PATH] [--no-pr] [--yolo] [--dry-run]")
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

	gov := &Governor{
		WindowDur:      parseDurDefault(f.Get("budget-window"), 24*time.Hour),
		WindowTokens:   int64(atoiDefault(f.Get("window-tokens"), 0)),
		Idle:           parseDurDefault(f.Get("idle"), 30*time.Minute),
		MaxCycles:      atoiDefault(f.Get("max-cycles"), 0),
		NoProgressHalt: atoiDefault(f.Get("no-progress-halt"), 3),
		StopFile:       resolveStopFile(w, f.Get("stop-file")),
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

type driver struct {
	ctx         *clikit.Ctx
	w           *workspace.Workspace
	cfg         loopCfg
	gov         *Governor
	run         runner
	sleep       func(time.Duration)
	now         func() time.Time
	trunkBranch string // the branch ship/integrate lands into; resolved once
}

func (d *driver) logf(format string, a ...any) {
	fmt.Fprintf(d.ctx.Stdout, format+"\n", a...)
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

	for {
		ready, err := readyTasks(d.w, d.cfg.project)
		if err != nil {
			return err
		}
		dec, why := d.gov.Before(len(ready), d.now())
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
			d.reviewPhase()
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
		landed := curTrunk - prevTrunk
		if landed < 0 {
			landed = 0
		}
		prevTrunk = curTrunk

		dec, why = d.gov.AfterCycle(landed, tokens)
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

	// BUILD — one detached implementer per task, each opening its own PR.
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
		}
	}

	// TEST — block until the detached wave finishes and finalizes.
	d.logf("  waiting on the wave…")
	d.run.run("wait", "wait")

	// LAND — two models, chosen by --pr:
	if d.cfg.pr {
		// Self-PR: each fixer opened its own PR and queued GitHub auto-merge
		// (dacli pr --auto), so GitHub lands it on green CI without the loop
		// re-integrating (re-opening a PR on an existing branch would only error).
		// The loop closes every built task's record here — otherwise the next
		// cycle re-picks a still-open task and reworks it — then commits the
		// workspace state. Whether a PR ACTUALLY merged is tracked separately by
		// trunk advancement in loop(), so closing the record never inflates the
		// thrash-guard's progress signal.
		d.logf("  closing built tasks; their PRs auto-merge on green CI…")
		for _, t := range batch {
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
		d.git("fetch", "-q", "origin", b) // best-effort; ignored when offline/no remote
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

func (d *driver) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = d.w.Root
	out, err := cmd.CombinedOutput()
	return string(out), err
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
	d.run.run("review", "spawn", "--task", ref, "--role", d.cfg.reviewRole)
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
		Context:  "Standing anchor for the autonomous review phase. Survey the code, tests, CI, and open findings; identify the ONE highest-value improvement grounded in evidence (a failing test, a reviewer finding, a real defect); `dacli task add` it with concrete acceptance criteria. Do NOT implement it here, and do NOT invent speculative work.",
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
