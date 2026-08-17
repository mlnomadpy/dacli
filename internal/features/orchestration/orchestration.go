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
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "loop", Usage: loopUsage, Brief: "Run the whole team process as a governed perpetual loop: review→plan→implement→test→land→retro, then repeat (--dry-run to preview, --max-cycles to bound). Token vocabulary: --max-tokens caps ONE cycle's spend, --window-tokens caps a rolling window, --token-window sets that window's duration (alias: --budget-window); --brief-tokens is the brief's SIZE, not spend", Mutates: true, Run: cmdLoop},
	{Path: "loop status", Brief: "Show the running/last loop's cycle count, trunk marker, tokens spent this window, and ready backlog size", Usage: "dacli loop status --project <slug>", Run: cmdLoopStatus},
}

// runner executes a dacli subcommand. Real runs shell out to this very binary
// so each phase is a logged, attributable run; tests inject a fake.
type runner interface {
	run(label string, args ...string) (string, error)
}

type resultRunner interface {
	runResult(label string, result any, args ...string) (string, error)
}

// execRunner invokes os.Executable() with the given args, inheriting the
// environment (so DACLI_AGENT identity flows into children).
type execRunner struct {
	cwd string
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

func (r execRunner) runResult(label string, result any, args ...string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		exe = "dacli"
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = r.cwd
	out, err := commandresult.Capture(cmd, result)
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
	project  string
	implRole string
	// implRoleExplicit distinguishes an operator's --impl-role decision from
	// the project-stack fallback. Only the latter may be replaced by automatic
	// cheapest-capable routing (task 373).
	implRoleExplicit bool
	reviewRole       string
	width            int   // implementers spawned per cycle
	perCycleTok      int64 // --max-tokens passed to each spawn (0 = unset)
	workerTimeout    int   // explicit --worker-timeout seconds (0 = derive from task estimate)
	dryRun           bool
	yolo             bool   // no between-cycle checkpoint pause
	pr               bool   // land through PRs + auto-merge
	into             string // --into: the branch ship/integrate land onto ("" = resolve)
	landing          model.LandingPolicy
	landingExplicit  bool
}

const (
	minimumWorkerTimeout    = 5 * time.Minute
	workerTimeoutPerTePoint = 5 * time.Minute
)

// workerTimeout returns the wall-clock allowance for one loop worker. An
// explicit loop flag is a policy override. Otherwise each expected estimate
// point buys five minutes, with the historical five-minute allowance retained
// as the floor for unestimated and sub-point tasks.
func (d *driver) workerTimeout(t *store.Task) int {
	if d.cfg.workerTimeout > 0 {
		return d.cfg.workerTimeout
	}
	timeout := minimumWorkerTimeout
	if t != nil {
		if tp, ok := t.Estimate(); ok {
			timeout = time.Duration(math.Ceil(tp.Expected()*workerTimeoutPerTePoint.Seconds())) * time.Second
			if timeout < minimumWorkerTimeout {
				timeout = minimumWorkerTimeout
			}
		}
	}
	return int(timeout / time.Second)
}

// loopUsage is the single source of truth for loop's flag synopsis: `--help`
// prints it and the missing-project usage error quotes it, so the two cannot
// drift. Every flag that TAKES A VALUE shows its value here — that is the
// whole point. `--no-progress-halt` requires an integer, appeared nowhere in
// help output, and reading it as a boolean was the only conclusion a user
// could reach (issue #421).
const loopUsage = "dacli loop --project <slug> [--width N] [--impl-role R] [--review-role R] " +
	"[--max-cycles N] [--window-tokens N --token-window DUR] [--max-tokens N] [--worker-timeout SEC] [--brief-tokens N] " +
	"[--idle DUR] [--halt-after-idle N] [--into BRANCH] [--stop-file PATH] [--no-pr] [--yolo] [--dry-run] [--advise]"

func cmdLoop(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("project", "impl-role", "review-role", "width", "max-tokens",
		"worker-timeout",
		"dry-run", "yolo", "pr", "no-pr", "advise", "budget-window", "window-tokens",
		"idle", "max-cycles", "no-progress-halt", "halt-after-idle", "into", "stop-file",
		"token-window", "brief-tokens"); err != nil {
		return err
	}

	project := f.Get("project")
	if project == "" {
		// Default to the sole project if there is exactly one.
		ps, _ := store.ListProjects(w)
		if len(ps) == 1 {
			project = ps[0].Slug
		} else {
			return clikit.Usagef("usage: %s", loopUsage)
		}
	}

	// Role defaults follow the project's recorded stack (dacli 192). They used
	// to be the constants "fixer" and "go-auditor", so a Python project was
	// audited by a role named for a language it does not use — observed live.
	// An explicit --impl-role/--review-role still wins over everything.
	stack := loopStack(w, project)
	inRoster := func(name string) bool { _, ok := store.LoadRole(w, name); return ok }

	// Every numeric/duration knob is read through the refusing clikit readers.
	// The hand-rolled helpers these replaced returned the DEFAULT on a parse
	// error, which meant `--window-tokens garbage` became 0 — read downstream as
	// "unlimited" — so an operator who asked for a cap ran with none, and
	// `--window-tokens 50k` silently became a 50-token ceiling.
	width, err := f.Int("width", 2)
	if err != nil {
		return err
	}
	perCycleTok, err := f.Int64("max-tokens", 0)
	if err != nil {
		return err
	}
	workerTimeout, err := f.Int("worker-timeout", 0)
	if err != nil {
		return err
	}
	if workerTimeout < 0 {
		return clikit.Usagef("--worker-timeout must be zero or a positive number of seconds (got %d)", workerTimeout)
	}
	// token-window is canonical; budget-window is the accepted old spelling.
	// They name a DURATION, which is why "budget" was a bad word for it: the
	// same root meant the brief's size elsewhere (task 292).
	windowFlag := "token-window"
	if n, ok := f.Alias("token-window", "budget-window"); ok {
		windowFlag = n
	}
	windowDur, err := f.Duration(windowFlag, 24*time.Hour)
	if err != nil {
		return err
	}
	windowTokens, err := f.Int64("window-tokens", 0)
	if err != nil {
		return err
	}
	idle, err := f.Duration("idle", 30*time.Minute)
	if err != nil {
		return err
	}
	maxCycles, err := f.Int("max-cycles", 0)
	if err != nil {
		return err
	}
	// --halt-after-idle is the canonical spelling. The original name asks the
	// reader to hold a double negative — "--no-progress-halt 2" reads as "do
	// not halt", so supplying a number feels wrong even though it is required
	// (issue #421). The old name keeps working: scripts already pass it.
	noProgressHalt, err := f.IntAliased(3, "halt-after-idle", "no-progress-halt")
	if err != nil {
		return err
	}

	cfg := loopCfg{
		project:          project,
		implRole:         orDefault(f.Get("impl-role"), prompts.RoleFor(stack, "fixer", "fixer", inRoster)),
		implRoleExplicit: f.Get("impl-role") != "",
		reviewRole:       orDefault(f.Get("review-role"), prompts.RoleFor(stack, "auditor", "go-auditor", inRoster)),
		width:            width,
		perCycleTok:      perCycleTok,
		workerTimeout:    workerTimeout,
		dryRun:           f.Bool("dry-run"),
		yolo:             f.Bool("yolo"),
	}
	journal, journalWarn := readCycleJournal(w, project)
	landing, landingExplicit, err := resolveLoopLanding(w, project, f, journal)
	if err != nil {
		return err
	}
	cfg.landing, cfg.landingExplicit = landing, landingExplicit
	cfg.pr = landing.Mode == model.LandingPR
	cfg.into = landing.Base

	// Validate --into UP FRONT. The branch is threaded into every ship and
	// integrate call, so a typo would otherwise surface deep inside a cycle
	// that has already spawned agents and spent tokens — and `ship --into` on a
	// branch that does not exist fails in a way that reads as a git problem
	// rather than a flag problem.
	if cfg.into != "" {
		if _, err := gitx.Run(w.Root, "rev-parse", "--verify", "--quiet", "refs/heads/"+cfg.into); err != nil {
			return clikit.Usagef("--into %s: no such local branch — create it first (git branch %s), or drop --into to land on the resolved trunk", cfg.into, cfg.into)
		}
	}

	// An explicit --pr with no remote cannot work: every landing check would
	// dead-end on gh, and nothing would ever close. Refuse rather than run a
	// loop that can only thrash (issue #382).
	if cfg.pr && !hasOriginRemote(w.Root) {
		return clikit.Refusedf("effective landing policy is pr, but this repo has no `origin` remote; add one and authenticate `gh`, or explicitly override with --no-pr (the task remains open)")
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
		WindowDur:      windowDur,
		WindowTokens:   windowTokens,
		Idle:           idle,
		MaxCycles:      maxCycles,
		NoProgressHalt: noProgressHalt,
		StopFile:       resolveStopFile(w, f.Get("stop-file")),
	}
	// A token ceiling with a zero-length window is not a ceiling: the window
	// rolls on every check and the spend resets before it is ever compared, so
	// the budget silently disables itself — and `--budget-window 0` parses
	// cleanly, so this is a plain flag combination, not an exotic one. Refuse
	// it rather than run an operator who asked to be capped uncapped (dacli
	// 218). With no --window-tokens the window is meaningless and this is fine.
	if gov.WindowTokens > 0 && gov.WindowDur <= 0 {
		return clikit.Usagef("--window-tokens %d needs a positive --budget-window (got %q): a zero-length window resets the spend before it is ever compared, which silently disables the budget",
			gov.WindowTokens, f.Get("budget-window"))
	}

	// A perpetual loop runs as a fresh process every checkpoint (the default,
	// non-yolo path returns after each cycle for the operator to re-run) — so
	// without this reload every restart would silently forget tokens already
	// spent this window and cycles/thrash-streak already accumulated, and a
	// --window-tokens or --no-progress-halt guard would never actually trip.
	// A snapshot that exists but does not parse is a REFUSAL, never a fresh
	// start: resuming from zeroes is exactly the state that clears the token
	// ceiling and the thrash streak, and the file sits in a repo the loop's own
	// children can write (dacli 207). The operator inspects and removes it.
	var restored governorState
	var restoredOK bool
	recovery := ""
	switch st, err := readGovernorState(w, project); {
	case err == nil:
		gov.Restore(st)
		// Governor snapshots written before task 379 do not carry a trunk
		// marker. The companion loop snapshot does, so use it to migrate a
		// persisted thrash halt instead of forcing an operator to discard the
		// otherwise-valid cycle and token-window accounting.
		if !st.TrunkMarkerKnown {
			if prior, priorErr := readLoopState(w, project); priorErr == nil && prior.Status == Halt.String() && strings.Contains(prior.Reason, "thrash guard tripped") {
				st.TrunkMarker = prior.TrunkMarker
				st.TrunkMarkerKnown = true
			}
		}
		restored, restoredOK = st, true
	case errors.Is(err, errCorruptState):
		return clikit.Refusedf("%v — refusing to resume with reset guards; inspect it, then delete %s to start a fresh window", err, governorStateFile(w, project))
	case errors.Is(err, os.ErrNotExist):
		if prior, priorErr := readLoopState(w, project); priorErr == nil && prior.Status == Halt.String() && strings.Contains(prior.Reason, "thrash guard tripped") {
			recovery = "explicit operator reset (governor state removed)"
		}
	}

	// A perpetual loop with no bound and no kill switch is a footgun. Require
	// one explicit termination affordance unless the operator opts into --yolo.
	if gov.MaxCycles == 0 && gov.NoProgressHalt == 0 && !cfg.yolo {
		return clikit.Usagef("refusing an unbounded loop with no stop condition: set --max-cycles N, keep --halt-after-idle > 0, or pass --yolo to accept a truly perpetual run (kill it with the stop file: %s)", gov.StopFile)
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

	d := &driver{ctx: ctx, w: w, cfg: cfg, gov: gov, run: run, sleep: time.Sleep, now: time.Now, recovery: recovery}
	if restoredOK {
		d.restoredTrunkMarker = restored.TrunkMarker
		d.restoredTrunkMarkerKnown = restored.TrunkMarkerKnown
	}

	// Resume the landing ledger the previous invocation checkpointed. Without
	// this the loop's three landing guarantees hold only in --yolo: the default
	// mode returns at every checkpoint, so in-memory pending state never
	// survived to the invocation that needed it (see journal.go).
	j := journal
	for _, msg := range journalWarn {
		// Say what was dropped. A silently shortened ledger looks exactly like
		// "nothing was outstanding", which is the failure this file prevents.
		fmt.Fprintf(ctx.Stderr, "warning: cycle journal: %s — that entry is not being reconciled this cycle\n", msg)
	}
	d.pendingAccept, d.pendingLand = j.PendingAccept, j.PendingLand
	if len(j.PendingAccept) > 0 || len(j.PendingLand) > 0 {
		d.logf("resuming: %d task(s) awaiting merge confirmation, %d record branch(es) in flight",
			len(j.PendingAccept), len(j.PendingLand))
	}
	// The ceiling, unlike the spend, was never persisted: a restart that
	// omitted --window-tokens restored the spend and then ran UNCAPPED. An
	// explicit flag still wins, so an operator can raise or lower it.
	if gov.WindowTokens == 0 && j.WindowTokens > 0 {
		gov.WindowTokens = j.WindowTokens
		d.logf("resuming token ceiling %d from the journal (pass --window-tokens to change it)", j.WindowTokens)
	}

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
	fmt.Fprintf(ctx.Stdout, "rollup: %s\n", st.Rollup)
	for _, line := range st.Rollup.Recovery() {
		fmt.Fprintf(ctx.Stdout, "  → %s\n", line)
	}
	fmt.Fprintf(ctx.Stdout, "last: %s", st.Status)
	if st.Reason != "" {
		fmt.Fprintf(ctx.Stdout, " (%s)", st.Reason)
	}
	if !st.UpdatedAt.IsZero() {
		fmt.Fprintf(ctx.Stdout, " · updated %s", st.UpdatedAt.Format(time.RFC3339))
	}
	fmt.Fprintln(ctx.Stdout)
	if st.Recovery != "" {
		fmt.Fprintf(ctx.Stdout, "recovery: %s\n", st.Recovery)
	}
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
	if cfg.implRoleExplicit {
		fmt.Fprintln(ctx.Stdout, "  build role source: explicit --impl-role override (phase routing may replace it only when the phase refuses its kind)")
	} else {
		fmt.Fprintf(ctx.Stdout, "  build role source: automatic cost routing per task (project-stack fallback %s; phase routing takes precedence when gated)\n", cfg.implRole)
	}

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
	ctx                      *clikit.Ctx
	w                        *workspace.Workspace
	cfg                      loopCfg
	gov                      *Governor
	run                      runner
	sleep                    func(time.Duration)
	now                      func() time.Time
	trunkBranch              string // the branch ship/integrate lands into; resolved once
	lastTrunkMarker          int    // most recently observed trunkMarker(), for status snapshots
	lastTrunkKnown           bool
	restoredTrunkMarker      int
	restoredTrunkMarkerKnown bool
	recovery                 string
	lastRollup               cycleRollup     // most recently computed cycle rollup, for status snapshots (dacli 299)
	pendingLand              []string        // self-PR branches opened this run not yet confirmed merged (see recordSelfPR)
	pendingAccept            []pendingAccept // built tasks whose `accept --force` awaits PR-merge confirmation (see reconcilePendingAccepts)
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
	Seq            int
	Branch         string
	Generation     int  // task generation whose landing this entry recovers (issue #679)
	GenerationSet  bool // false only for backward-compatible legacy journal entries
	VerifyRequired bool // recovery already surfaced the owner-only --verify action (issue #661)
}

func (d *driver) logf(format string, a ...any) {
	fmt.Fprintf(d.ctx.Stdout, format+"\n", a...)
}

// saveState persists a status snapshot for `dacli loop status` to read — best
// effort, called at every governor checkpoint.
func (d *driver) saveState(status, reason string, backlog int) {
	// A preview must not become the last production outcome or consume any of
	// the governor/journal state it is describing (task 370). Keeping the guard
	// here makes every present and future checkpoint dry-run safe.
	if d.cfg.dryRun {
		return
	}
	// The landing ledger rides every checkpoint, because the default mode
	// RETURNS at each one: without this, `pendingAccept`/`pendingLand` die with
	// the process and the next invocation re-picks tasks whose PRs merged and
	// pushes the record out from under PRs still in flight (see journal.go).
	writeCycleJournal(d.w, d.cfg.project, cycleJournal{
		PendingAccept:   d.pendingAccept,
		PendingLand:     d.pendingLand,
		WindowTokens:    d.gov.WindowTokens,
		Landing:         d.cfg.landing,
		LandingExplicit: d.cfg.landingExplicit,
	})
	writeLoopState(d.w, loopState{
		Project:      d.cfg.project,
		Cycle:        d.gov.Cycle(),
		TrunkMarker:  d.lastTrunkMarker,
		WindowTokens: d.gov.WindowSpent(),
		Backlog:      backlog,
		Status:       status,
		Reason:       reason,
		Recovery:     d.recovery,
		Rollup:       d.lastRollup,
		UpdatedAt:    d.now(),
	})
	govState := d.gov.State()
	govState.TrunkMarker = d.lastTrunkMarker
	govState.TrunkMarkerKnown = d.lastTrunkKnown
	writeGovernorState(d.w, d.cfg.project, govState)
}

func (d *driver) loop() error {
	d.logf("dacli loop — project %s · impl=%s · review=%s · width=%d%s",
		d.cfg.project, d.cfg.implRole, d.cfg.reviewRole, d.cfg.width, dryTag(d.cfg.dryRun))
	action, gates := "local merge", "task acceptance and configured verification"
	if d.cfg.pr {
		action, gates = "open/reuse PR and queue auto-merge", "GitHub required checks and reviews"
	}
	d.logf("landing policy: mode=%s · base=%s · override=%t · PR action=%s · required gates=%s",
		d.cfg.landing.Mode, clikit.OrDash(d.trunkBase(), "repository default"), d.cfg.landingExplicit, action, gates)
	if d.gov.MaxCycles > 0 {
		d.logf("bounded to %d cycle(s); stop file: %s", d.gov.MaxCycles, d.gov.StopFile)
	} else {
		d.logf("perpetual; stop file: %s · thrash-halt after %d cycles with no trunk advance", d.gov.StopFile, d.gov.NoProgressHalt)
	}

	d.trunkBranch = d.resolveTrunkBranch()
	if d.trunkBranch == "" {
		d.logf("note: no trunk branch could be resolved (detached HEAD with no main/master?) — trunk measurement degrades to a best-effort default")
	}
	// prevTrunkKnown carries "we have a real baseline to subtract from". Until
	// one measurement succeeds there is nothing to compare against, and a
	// missing baseline must never be spelled 0 — see trunkMarker (dacli 212).
	prevTrunk, prevTrunkKnown := d.trunkMarker()
	if prevTrunkKnown {
		d.lastTrunkMarker = prevTrunk
		d.lastTrunkKnown = true
		if d.restoredTrunkMarkerKnown && prevTrunk > d.restoredTrunkMarker {
			d.gov.ResetZeroStreak()
			d.recovery = fmt.Sprintf("observed trunk advanced between invocations (%d → %d)", d.restoredTrunkMarker, prevTrunk)
			d.logf("recovery: %s", d.recovery)
		}
	}

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
		// see pendingAccept and reconcilePendingAccepts. Its rollup contribution (a
		// merge or an orphan observed THIS pass, however long ago the task was
		// built) seeds this cycle's tally; runCycle's own batch classification
		// below is added to it once — and if — a cycle actually runs (dacli 299).
		var reconcileRollup cycleRollup
		if !d.cfg.dryRun {
			reconcileRollup = d.reconcilePendingAccepts()
		}
		d.lastRollup = reconcileRollup

		// Reclaim worktrees whose branch has already landed or whose run is
		// finished. reconcilePendingAccepts/gcBranch only reap the branches THIS
		// loop is tracking a PR for; a task closed by an operator, a prior loop,
		// or a --worktree spawn outside the accept flow leaves its checkout
		// behind forever — one live checkout per task ever spawned, until a real
		// run hit 86 worktrees / 2.2 GB (dacli 252). This blanket sweep is the
		// safety-checked catch-all (see store.ReclaimableWorktrees).
		if !d.cfg.dryRun {
			d.reapWorktrees()
		}

		// Walk the stage gates as far as they open before choosing this
		// cycle's work — a project sitting in a phase whose gates have all
		// passed is deadlock, not process (see advanceStages, dacli 189).
		if !d.cfg.dryRun {
			d.advanceStages()
		}

		ready, err := d.readyTasks()
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
			if d.cfg.dryRun {
				return nil
			}
			d.gov.ChargeIdleTokens(store.RunsTokensSince(d.w, since))
			d.saveState(dec.String(), why, len(ready))
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

		tokens, batchRollup := d.runCycle(ready)
		d.lastRollup = reconcileRollup.add(batchRollup)
		d.logf("  cycle rollup: %s", d.lastRollup)
		for _, line := range d.lastRollup.Recovery() {
			d.logf("    → %s", line)
		}
		// The preview has now printed the complete action plan. Do not measure it
		// as a production cycle: AfterCycle would increment cycle/streak/window
		// state and can falsely trip a thrash halt one step below the threshold.
		if d.cfg.dryRun {
			d.logf("(dry-run: one cycle previewed; stopping)")
			return nil
		}

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
		//
		// A cycle whose marker could not be READ is charged as unmeasured: the
		// thrash streak is left exactly as it was and prevTrunk keeps its last
		// good value, so the next successful measurement computes a real delta
		// spanning both cycles instead of one fabricated zero followed by one
		// fabricated repo-sized jump (dacli 212).
		landed := 0
		measured := false
		if curTrunk, ok := d.trunkMarker(); ok {
			d.lastTrunkMarker = curTrunk
			d.lastTrunkKnown = true
			if prevTrunkKnown {
				landed = curTrunk - prevTrunk
				if landed < 0 {
					landed = 0
				}
				measured = true
			}
			prevTrunk, prevTrunkKnown = curTrunk, true
		} else {
			d.logf("  note: could not measure trunk on %s — this cycle counts as unmeasured, not as zero progress", orDefault(d.trunkBranch, "main"))
		}

		if measured {
			dec, why = d.gov.AfterCycle(landed, tokens)
		} else {
			dec, why = d.gov.AfterCycleUnmeasured(tokens)
		}
		remaining, _ := readyTasks(d.w, d.cfg.project)
		d.saveState(dec.String(), why, len(remaining))
		if dec == Halt {
			d.logf("● halt: %s", why)
			return nil
		}
		// The stop file is re-checked here as well as in Before(): a whole wave
		// of child agents ran since the last check, and any of them (or the
		// operator watching them) may have asked the loop to stop (dacli 207).
		if d.gov.StopRequested() {
			d.saveState(Halt.String(), d.gov.StopReason(), len(remaining))
			d.logf("● halt: %s", d.gov.StopReason())
			return nil
		}
		if !d.cfg.yolo {
			progress := fmt.Sprintf("trunk advanced by %d", landed)
			if !measured {
				progress = "trunk advance not measurable this cycle"
			}
			d.logf("— cycle %d done (%s). Checkpoint: re-run to continue, or touch %s to stop —",
				d.gov.Cycle(), progress, d.gov.StopFile)
			return nil
		}
	}
}

// runCycle executes one full sprint: build → test → sync → land → review →
// retro. It returns the tokens charged and a rollup of how the batch
// resolved (landed / produced nothing / stalled / blocked — see
// classifyBatch); trunk-advancement (the thrash-guard signal) is measured by
// the caller across the cycle, not derived from a task-status delta here —
// see loop().
func (d *driver) runCycle(ready []*store.Task) (tokens int64, rollup cycleRollup) {
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
	// The fallback role is resolved per cycle, not read straight off the
	// config: on a phase-gated project the configured implementer may have no
	// work in the current phase, and spawning it anyway is a guaranteed
	// refusal (dacli 189). On an untemplated project buildRole returns
	// cfg.implRole unchanged.
	fallbackRole := d.buildRole()
	fallbackSource := "automatic cost routing"
	if d.cfg.implRoleExplicit {
		fallbackSource = "explicit override"
	}
	if fallbackRole != d.cfg.implRole {
		fallbackSource = "phase routing"
	}
	// Within the fallback's kind, each task then routes to the cheapest role
	// whose capacity covers ITS OWN Te — the loop used to spawn every task in
	// the batch with the one fallback role regardless of size, so a one-line
	// typo fix and a subsystem rewrite landed on the same (often the most
	// expensive) model. team.CheapestCapable already existed for `dacli team
	// assign`/`spawn --advise` (230, 231); this wires the loop's own batching
	// through it. A task with no estimate, or a roster with nothing else of
	// that kind, still spawns with the fallback exactly as before (dacli 233).
	// Size anything unsized BEFORE routing and ranking read the estimates.
	// Both silently degrade without one — CheapestCapable is skipped (:672)
	// and haveCPM drops to MoSCoW (:1761) — so an unestimated backlog quietly
	// loses the two orderings the loop appears to be using.
	d.sizeUnestimated(batch)
	// Sizing is best-effort, so re-read the batch and say plainly when it did
	// not take. A task that is STILL unsized will be refused by every capped
	// role, so this cycle is structurally unable to build it — and the thrash
	// guard would otherwise report only "no net progress", leaving the cause
	// buried in a per-spawn refusal above it (issue #430, suggestion 3).
	d.reportStillUnsized(batch)
	// Resolve the whole wave's enforcement boundaries before launching any of
	// it. A live-only collision gate is too late: the second task is refused
	// after the first spawn and looks like ordinary retryable failure. Planning
	// the claims once also makes dry-run and live execution use identical input.
	plannedClaims := make(map[int][]string, len(batch))
	var claimed []string
	for _, t := range batch {
		claims := store.ClaimHints(d.w.Root, t)
		if theirs, mine, overlap := procmon.PathsOverlap(claimed, claims); overlap {
			d.logf("  → %03d: planned claim collision (%s overlaps %s) — leaving open without spawning", t.Seq, mine, theirs)
			continue
		}
		plannedClaims[t.Seq] = claims
		claimed = append(claimed, claims...)
	}

	roles, _ := store.LoadRoles(d.w)
	calibration := store.LoadCalibration(d.w)
	outcomes := store.FirstPassOutcomes(d.w)
	fallbackKind := ""
	if role, ok := store.LoadRole(d.w, fallbackRole); ok {
		fallbackKind = role.Kind
	}
	for _, t := range batch {
		claims, planned := plannedClaims[t.Seq]
		if !planned {
			continue
		}
		// The stop file is re-checked before EVERY spawn, not once per cycle in
		// Before(): a wave is the longest stretch of the loop, it is where all
		// the tokens go, and an operator who touches STOP while agents are
		// running means "launch no more", not "one more full width of them"
		// (dacli 207).
		if d.gov.StopRequested() {
			d.logf("  ● %s — launching no further agents this wave", d.gov.StopReason())
			break
		}
		buildRole := fallbackRole
		var routing team.Explanation
		if fallbackSource == "automatic cost routing" && fallbackKind != "" {
			if tp, sized := t.Estimate(); sized {
				candidates := d.routeCandidates(roles, calibration.Samples, outcomes, fallbackRole, fallbackKind, tp.Expected())
				routing = (team.Strategy{}).Select(team.RouteRequirements{Kind: fallbackKind, Grant: "rw", Title: t.Title, Paths: t.PathHints(), TaskPoints: tp.Expected(), TokenBudget: float64(d.cfg.perCycleTok)}, candidates)
				if routing.Selected.Role != "" {
					buildRole = routing.Selected.Role
				}
			}
		}
		if routing.Selected.Role == "" {
			if role, ok := store.LoadRole(d.w, buildRole); ok {
				routing.Selected = team.RouteSelection{Role: role.Name, Runtime: role.Runtime, Model: role.ModelID()}
			}
		}
		if !d.cfg.dryRun {
			writeRoutingExplanation(d.w, cycle, t.Seq, routing)
		}
		ref := t.ID
		spawn := []string{"spawn", "--task", ref, "--role", buildRole, "--detach", "--worktree"}
		// Claim the task's own path hints (dacli 299): the loop used to spawn
		// every wave with no --claim at all, so `gateClaimOverlap` had nothing to
		// arbitrate and two tasks touching the same tree could run in parallel and
		// merge-conflict each other — the operator did that arbitration by hand
		// (see the a-root finding this task was filed from). A claim carrying no
		// paths (a task whose text names nothing path-like) omits the flag
		// entirely, matching splitClaims's own "no claim" behavior.
		// ClaimHints, not PathHints: a claim is an ENFORCEMENT boundary, and
		// PathHints is documented as crude because for routing a spurious
		// token costs one weak tie-break vote. Here it cost an agent its whole
		// commit — task 338's "G104/G301/G302/G306" became its claim and
		// eighteen legitimate files were refused (issue #427). Only tokens
		// that resolve to a real path in the repo become a claim.
		if claim := strings.Join(claims, ","); claim != "" {
			spawn = append(spawn, "--claim", claim)
		}
		if d.cfg.pr {
			spawn = append(spawn, "--pr")
		}
		if d.cfg.perCycleTok > 0 {
			spawn = append(spawn, "--max-tokens", fmt.Sprint(d.cfg.perCycleTok))
		}
		spawn = append(spawn, "--timeout", fmt.Sprint(d.workerTimeout(t)))
		d.logf("  → %s: %s — role %s (%s)", ref, t.Title, buildRole, fallbackSource)
		if out, err := d.run.run("spawn", spawn...); err != nil {
			d.logf("    spawn refused/failed: %s", clikit.FirstLine(out))
			continue
		}
		built[t.Seq] = true
	}

	// TEST — block until the detached wave finishes and finalizes.
	d.logf("  waiting on the wave…")
	// A failed wait is not cosmetic: everything after this point — sync,
	// accept, integrate — assumes the wave FINISHED. Proceeding silently on a
	// failure means acting on half-written work, so say so. It is reported
	// rather than fatal because the steps below re-derive state from disk and
	// will simply find less to do.
	if out, err := d.run.run("wait", "wait"); err != nil {
		d.logf("    wait failed (%v) — the steps below assume the wave finished, so treat this cycle's results as partial: %s",
			err, clikit.FirstLine(out))
	}
	for _, t := range batch {
		if built[t.Seq] && d.policyRefusedSince(t.ID, since) {
			d.logf("    %03d: child ended in exit-3 policy refusal — blocking instead of retrying unchanged", t.Seq)
			if cur, err := store.FindTask(d.w, fmt.Sprintf("%03d", t.Seq)); err == nil {
				store.AppendLog(cur, "blocked after an exit-3 policy refusal; change the claim or policy before retrying")
				_ = store.SaveTask(cur)
				_ = store.MoveTask(d.w, cur, model.StatusBlocked)
			}
			built[t.Seq] = false
		}
	}

	// SYNC — apply every pending proposal a read-only agent in the wave filed
	// as an event (a status change via `task block`/`task done`, a finding) so
	// it lands in the objects it references BEFORE this cycle's LAND step and
	// before the caller judges whether the cycle produced anything (dacli
	// 299). A read-only grant is the DEFAULT for a spawned agent — it cannot
	// mutate a task it does not own, so its "done" or "blocked" only ever
	// reaches the event log until the owner applies it; without this the loop
	// itself never called `sync`, so that work sat pending forever and the
	// task it touched read as untouched. Run before LAND so any file changes
	// sync makes ride in the SAME cycle's record commit, not an uncommitted
	// tree left for the next cycle to trip over.
	if out, err := d.run.run("sync", "sync"); err != nil {
		d.logf("  sync: %s", clikit.FirstLine(out))
	}

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
			d.pendingAccept = append(d.pendingAccept, pendingAccept{Seq: t.Seq, Branch: branch, Generation: t.Generation(), GenerationSet: true})
			d.pendingLand = append(d.pendingLand, branch)
		}
		d.recordSelfPR()
	} else {
		// Local model: fixers committed to their branches without opening PRs, so
		// the loop integrates them into trunk itself.
		d.logf("  integrating done branches…")
		// The local-landing path. A silent failure here is precisely issue
		// #419's shape: branches never reach trunk, tasks never close, and the
		// cycle reports "trunk advanced by 0" with no cause attached — the one
		// number the reporter said was their only symptom.
		if out, err := d.run.run("ship", d.shipArgs("--project", d.cfg.project)...); err != nil {
			d.logf("    integrate failed — NOTHING landed on trunk this cycle: %s", clikit.FirstLine(out))
		}
	}

	// ROLLUP — classify how this cycle's batch resolved (dacli 299): landed,
	// produced nothing, still in flight (stalled), or blocked. Computed here,
	// after LAND and after the SYNC step above has folded in anything a
	// read-only build agent proposed, so a task a wave blocked on a question
	// reports as blocked rather than as an ordinary in-flight task.
	rollup = d.classifyBatch(batch, built)

	// REVIEW — regenerate the backlog: an auditor files the next
	// evidence-based improvement(s) as fresh tasks. Skipped once STOP is
	// present: landing what the wave already produced is finishing started
	// work, but the review spawn is a NEW agent, and the stop file's promise is
	// that no new agent starts after it appears (dacli 207).
	if d.gov.StopRequested() {
		d.logf("  ● %s — skipping the review spawn", d.gov.StopReason())
		return
	}
	d.reviewPhase(batch...)

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
	// Best-effort: the retro note is a record, and losing one must never stop a
	// cycle that has already landed work. Explicitly discarded so the next
	// reader knows it is a decision rather than an oversight.
	_, _ = d.run.run("retro", "retro", d.cfg.project, "--improve",
		fmt.Sprintf("cycle: %d of %d spawned task(s) produced work; follow-ups are filed as tasks by the review phase", builtCount, len(batch)))

	// Workspace health, once per cycle. The loop never ran doctor, so duplicate
	// tasks, orphaned records and unparseable task files stayed invisible on
	// the ONE path nobody is watching — the inversion the audit for task 300
	// named: the unattended run should be the best-covered, not the least.
	//
	// Reported, never fatal: a corrupt workspace is something an operator must
	// see, but halting a governed loop on it would trade a visible problem for
	// a stalled machine.
	d.reportWorkspaceHealth()

	// The deferred token charge above sums every run this cycle produced
	// (build spawns + the review spawn) from their usage.txt actuals — 0 for
	// any run whose runtime never reported usage, the same honest degrade
	// calibration applies elsewhere.
	return
}

func (d *driver) routeCandidates(roles []team.Role, samples []store.CalibSample, outcomes map[store.Band]store.FirstPassOutcome, sourceName, kind string, te float64) []team.RouteCandidate {
	limits := store.LoadRuntimeLimits(d.w)
	allowed := map[string]bool{}
	if source, ok := store.LoadRole(d.w, sourceName); ok && source.Runtime != "" {
		if _, paused, _ := limits.Open(source.Runtime); paused {
			allowed[source.Name] = true
			for _, name := range source.FallbackTo {
				allowed[name] = true
			}
		}
	}
	out := make([]team.RouteCandidate, 0, len(roles))
	for _, role := range roles {
		if !strings.EqualFold(role.Kind, kind) || (len(allowed) > 0 && !allowed[role.Name]) {
			continue
		}
		band := store.Band{Role: role.Name, Model: role.ModelID(), Runtime: role.Runtime}
		ratio, tokenN := store.MedianTokenRatio(samples, band)
		stat := outcomes[band]
		paused := false
		if role.Runtime != "" {
			_, paused, _ = limits.Open(role.Runtime)
		}
		// An undeclared grant retains the legacy role behavior; a declared ro
		// role is ineligible for the loop's implementation wave.
		grantEnforced := role.Grant != "ro"
		if role.Grant == "" {
			role.Grant = "rw"
		}
		if rt, err := store.LoadRuntime(d.w, role.Runtime); err == nil {
			grantEnforced = grantEnforced && store.RuntimeWritable(rt)
		}
		capacity := 1
		if role.WIP > 0 {
			if active, err := store.ActiveInRole(d.w, role.Name); err != nil {
				capacity = 0
			} else {
				capacity = role.WIP - active
			}
		}
		out = append(out, team.RouteCandidate{
			Role: role, GrantEnforced: grantEnforced, ContextLimit: role.Profile.ContextLimit,
			CapacityRemaining: capacity, RemainingBudget: float64(d.cfg.perCycleTok), ProviderPaused: paused,
			Metrics: team.RouteMetrics{TokensPerCompleted: ratio * te, TokenSamples: tokenN, FirstPassSuccess: stat.Rate, SuccessSamples: stat.Samples, LatencySeconds: medianBandHours(samples, band) * 3600},
		})
	}
	return out
}

func medianBandHours(samples []store.CalibSample, band store.Band) float64 {
	var values []float64
	for _, sample := range samples {
		if sample.Band == band {
			values = append(values, sample.Hours)
		}
	}
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	return values[len(values)/2]
}

func routingExplanationFile(w *workspace.Workspace, cycle, seq int) string {
	return filepath.Join(w.Root, workspace.Dir, "loop", "routing", fmt.Sprintf("cycle-%03d-task-%03d.json", cycle, seq))
}

func writeRoutingExplanation(w *workspace.Workspace, cycle, seq int, explanation team.Explanation) {
	raw, err := json.MarshalIndent(explanation, "", "  ")
	if err != nil {
		return
	}
	path := routingExplanationFile(w, cycle, seq)
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	_ = writeStateFile(path, string(append(raw, '\n')))
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
	// ship distinguishes the effective PR policy from selecting its PR-capable
	// execution path. A configured policy is intentionally not forwarded as a
	// CLI override by shipArgs, so the record tail must select that path itself
	// or ship refuses before writing the collaboration record (issue #663).
	if d.cfg.landing.Mode == model.LandingPR {
		args = append(args, "--pr")
	}
	if len(d.pendingLand) == 0 {
		args = append(args, "--push")
	} else {
		d.logf("  record: holding the push — %d PR(s) still in flight (%s); pushes once they land", len(d.pendingLand), strings.Join(d.pendingLand, ", "))
	}
	if out, err := d.run.run("record", args...); err != nil {
		d.logf("  record: ship failed: %s", clikit.FirstLine(out))
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
		_, _ = gitx.RunNetwork(d.w.Root, "fetch", "-q", "--prune", "origin")
	}
	still := branches[:0]
	for _, b := range branches {
		if _, err := d.git("rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+b); err != nil {
			continue // no remote ref at all: nothing can be in flight
		}
		// A remote ref is NOT a PR. A leftover origin/dacli/NNN — from an
		// attempt that never opened one, which nothing deletes — used to read
		// as "still in flight" forever, so the record push was held
		// indefinitely and every later cycle repeated the same conclusion.
		// Ask what the PR is actually doing; only an open one holds the push.
		switch d.prLandStatus(b) {
		case "landing", "stranded":
			still = append(still, b)
		case "merged", "orphaned":
			// Landed or dead — either way it is not blocking the record.
		default:
			// "unknown": gh unreachable or no PR found. Treat a bare ref with
			// no discoverable PR as NOT in flight, because the alternative is
			// the wedge this fixes — an indefinite hold on evidence that is
			// only the ref's existence.
			d.logf("    %s: a remote branch exists but no PR was found — not holding the record for it", b)
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
//
// The returned cycleRollup is this call's contribution to the cycle's rollup
// (dacli 299): a merge confirmed here is real trunk-landed work even though
// the task may have been BUILT a cycle or more ago, so it belongs in the
// rollup of the cycle that actually observed the landing, not silently
// dropped because runCycle's own batch never touched it this time around.
func (d *driver) reconcilePendingAccepts() cycleRollup {
	var r cycleRollup
	if len(d.pendingAccept) == 0 {
		return r
	}
	remaining := d.pendingAccept[:0]
	for _, p := range d.pendingAccept {
		task, taskErr := store.FindTask(d.w, fmt.Sprintf("%s/%03d", d.cfg.project, p.Seq))
		if taskErr == nil && ((!p.GenerationSet && task.Generation() > 0) || (p.GenerationSet && task.Generation() != p.Generation)) {
			// A reopen deliberately reuses the same sequence and branch for new
			// corrective work. Its earlier merged PR is not evidence that this
			// generation landed, and must neither exclude it nor trigger GC.
			d.logf("    %03d: task was reopened after this recovery entry — invalidating prior-generation pending accept", p.Seq)
			continue
		}
		switch d.prLandStatus(p.Branch) {
		case "merged":
			if taskErr != nil {
				d.logf("    %03d: PR merged but task state could not be resolved — keeping recovery entry: %s", p.Seq, taskErr)
				remaining = append(remaining, p)
				continue
			}
			if task.Status == model.StatusDone && acceptanceComplete(task) {
				// An owner may finish verification between bounded loop invocations.
				// Re-running accept here duplicated evidence and, for command criteria,
				// retried a policy refusal forever even though the canonical record was
				// already truthful (issue #661).
				d.logf("    %03d: PR merged and task already fully accepted — clearing stale recovery entry", p.Seq)
				d.gcBranch(p.Branch)
				continue
			}
			if taskRequiresVerifierEvidence(task) {
				if !p.VerifyRequired {
					d.logf("    %03d: PR merged but command acceptance requires verifier evidence — run `dacli accept %03d --verify \"<command>\"`; keeping the recovery entry", p.Seq, p.Seq)
				}
				p.VerifyRequired = true
				remaining = append(remaining, p)
				continue
			}
			d.logf("    %03d: PR merged — closing the task record", p.Seq)
			// The close must SUCCEED before this counts as landed. Discarding
			// the error meant a failed accept still incremented Landed and
			// still deleted the branch: the rollup reported the task as landed
			// while it sat open, and the branch that was the evidence was gone.
			// Record-disagrees-with-reality, plus the recovery path destroyed
			// in the same breath (found by errcheck during the dacli 336 review).
			if out, err := d.run.run("accept", "accept", fmt.Sprintf("%03d", p.Seq), "--force"); err != nil {
				d.logf("    %03d: PR merged but accept FAILED — task left open and its branch kept for recovery: %s",
					p.Seq, clikit.FirstLine(out))
				remaining = append(remaining, p)
				continue
			}
			d.gcBranch(p.Branch)
			r.Landed++
		case "orphaned":
			// "Fresh retry" was not fresh. Nothing removed the branch, and
			// AddWorktree reuses an existing one AT ITS OLD TIP, so the next
			// cycle rebuilt on the abandoned base, hit the same non-fast-
			// forward push, and reached the same conclusion — forever. Clear
			// the local branch and worktree, and the stale remote ref that
			// stillPending would otherwise keep reading.
			d.logf("    %03d: PR closed without merging — clearing the branch so the retry starts from trunk", p.Seq)
			d.gcBranch(p.Branch)
			d.dropRemoteBranch(p.Branch)
			r.ProducedNothing++
		case "awaiting-pr":
			// A successful empty PR query is not a closed PR. The agent may
			// have committed and pushed before PR creation failed (task 366
			// after run 01KZVR1TQH); keep both refs and tell the operator the
			// missing lifecycle step instead of destroying verified work.
			d.logf("    %03d: branch built and awaiting PR creation — keeping the branch for recovery", p.Seq)
			remaining = append(remaining, p)
			r.Stalled++
		case "stranded":
			// Open, but auto-merge never queued — it will NOT self-land. Say so
			// loudly instead of silently treating it like a queued PR: without this
			// a stranded PR sits open forever, counted as "still landing", holding
			// the record push back and never surfacing that no one is going to
			// merge it (task 290). Kept pending so the loop keeps watching (a human
			// may queue or merge it) rather than dropped, which would re-rank the
			// task and open a duplicate PR against the still-open one.
			d.logf("    %03d: PR open but NOT queued for auto-merge — it will NOT self-land; queue it (`gh pr merge %s --auto`) or merge it by hand (task 290)", p.Seq, p.Branch)
			remaining = append(remaining, p)
			r.Stalled++
		default: // "landing" (queued, PR still open) or "unknown" (gh/network unreachable)
			remaining = append(remaining, p)
			r.Stalled++
		}
	}
	d.pendingAccept = remaining
	return r
}

func acceptanceComplete(t *store.Task) bool {
	boxes := t.Acceptance()
	if len(boxes) == 0 {
		return false
	}
	for _, box := range boxes {
		if !box.Done {
			return false
		}
	}
	return true
}

func taskRequiresVerifierEvidence(t *store.Task) bool {
	for i := range t.Acceptance() {
		if store.AcceptanceRequiresCommandVerification(t, i+1) {
			return true
		}
	}
	return false
}

// cycleRollup is the per-cycle outcome tally `dacli loop status` surfaces
// (dacli 299): how many of the tasks the loop touched this cycle actually
// reached trunk, produced no work at all, are still in flight, or ended the
// cycle blocked — so an unattended run's health is legible from the
// persisted state file alone, without replaying its stdout log.
type cycleRollup struct {
	Landed          int // work reached trunk (a confirmed PR merge, or a local integrate)
	ProducedNothing int // spawn refused/failed, a branch with no commits, or a PR closed unmerged
	Stalled         int // built (or previously built) but not yet confirmed landed
	Blocked         int // the task ended the cycle in status blocked
}

// add returns the element-wise sum of r and o — combining reconcile's
// this-pass classification of PRIOR cycles' pending work with THIS cycle's
// own batch classification into the one rollup a checkpoint persists.
func (r cycleRollup) add(o cycleRollup) cycleRollup {
	return cycleRollup{
		Landed:          r.Landed + o.Landed,
		ProducedNothing: r.ProducedNothing + o.ProducedNothing,
		Stalled:         r.Stalled + o.Stalled,
		Blocked:         r.Blocked + o.Blocked,
	}
}

func (r cycleRollup) String() string {
	return fmt.Sprintf("landed %d · produced nothing %d · stalled %d · blocked %d",
		r.Landed, r.ProducedNothing, r.Stalled, r.Blocked)
}

// Recovery renders the NEXT STEP for each non-landing outcome, one line each,
// and nothing at all when everything landed.
//
// A count alone tells an operator that something went wrong and leaves them to
// open six transcripts to find out what to do about it — which is the work
// this rollup exists to replace (task 271). Each line names the command that
// answers "and now what?", so the rollup is a starting point rather than a
// verdict.
func (r cycleRollup) Recovery() []string {
	var out []string
	if r.ProducedNothing > 0 {
		out = append(out, fmt.Sprintf("%d produced nothing — the spawn failed or committed nothing: `dacli runs list` for the outcome, `dacli logs <run>` for why. The task stayed open and will be re-picked.", r.ProducedNothing))
	}
	if r.Stalled > 0 {
		out = append(out, fmt.Sprintf("%d stalled — built but not confirmed landed: `dacli pr status --task <ref>` says whether the PR is merging, behind base, or conflicted.", r.Stalled))
	}
	if r.Blocked > 0 {
		out = append(out, fmt.Sprintf("%d blocked — an agent asked a question it could not answer: `dacli threads` lists them, `dacli answer <id> \"...\"` unblocks the task.", r.Blocked))
	}
	return out
}

// classifyBatch tallies how each task in this cycle's build batch resolved,
// for the rollup (dacli 299). Called after LAND and after the SYNC step has
// applied any status a read-only build agent could only propose, so a task
// the wave blocked on a question (`dacli task block`/`ask`, applied by sync)
// reports blocked rather than merely in flight.
//
//   - ProducedNothing — the spawn never produced a commit at all: refused/
//     failed synchronously, or wait finished with an empty branch (built[t.Seq]
//     was cleared by the post-wait branchHasWork check above).
//   - Blocked — the task's CURRENT status is blocked.
//   - Landed — --no-pr only: ship's integrate step reached trunk and closed
//     the task this same cycle. Under --pr a task never lands within its own
//     build cycle (GitHub merges asynchronously); its landing is observed and
//     rolled up later by reconcilePendingAccepts.
//   - Stalled — everything else: a --pr build parked in pendingAccept awaiting
//     merge confirmation, or a --no-pr integrate that hit a conflict (blocked,
//     per docs/vcs, "never half-merges") and left the task open.
//
// A task that cannot even be reloaded is counted stalled, never landed or
// blocked — the same honest-degrade rule an unmeasurable trunk gets (dacli
// 212): absence of a signal must never be spelled as a stronger one.
func (d *driver) classifyBatch(batch []*store.Task, built map[int]bool) cycleRollup {
	var r cycleRollup
	for _, t := range batch {
		if !built[t.Seq] {
			if cur, err := store.FindTask(d.w, fmt.Sprintf("%03d", t.Seq)); err == nil && cur.Status == model.StatusBlocked {
				r.Blocked++
			} else {
				r.ProducedNothing++
			}
			continue
		}
		cur, err := store.FindTask(d.w, fmt.Sprintf("%03d", t.Seq))
		switch {
		case err != nil:
			r.Stalled++
		case cur.Status == model.StatusBlocked:
			r.Blocked++
		case !d.cfg.pr && cur.Status == model.StatusDone:
			r.Landed++
		default:
			r.Stalled++
		}
	}
	return r
}

func (d *driver) policyRefusedSince(taskID, since string) bool {
	entries, _ := os.ReadDir(d.w.RunsDir())
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() <= since {
			continue
		}
		dir := d.w.RunDir(entry.Name())
		rec, err := procmon.ReadRecord(filepath.Join(dir, "proc.txt"))
		if err != nil || rec.Task != taskID {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, "outcome.md"))
		if err == nil && strings.Contains(string(raw), "exit: exit status 3") {
			return true
		}
	}
	return false
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
				_ = gitx.RemoveWorktree(d.w.Root, wt.Path)
				break
			}
		}
	}
	_, _ = d.git("branch", "-D", branch)
}

// reapWorktrees is the blanket, safety-checked counterpart to gcBranch: once a
// cycle it reclaims EVERY worktree whose branch has landed on trunk or whose
// run has finished, not just the ones this loop is tracking a PR for. gcBranch's
// caution against a blanket ancestry sweep is honored inside
// store.ReclaimableWorktrees — a bare-tipped live spawn (zero commits, trivially
// an ancestor of trunk) is never reclaimed, and a merged worktree is touched
// only when its tree is clean. Best-effort: a reap failure never blocks the loop.
func (d *driver) reapWorktrees() {
	trunk := d.trunkBranch
	if trunk == "" {
		trunk = "main"
	}
	removed, err := store.PruneWorktrees(d.w, trunk)
	if err != nil {
		return
	}
	for _, c := range removed {
		d.logf("    reclaimed worktree %s (%s — %s)", filepath.Base(c.Path), c.Branch, c.Reason)
	}
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
//   - "landing"  — a PR is open WITH GitHub auto-merge queued; it lands itself
//     the instant CI passes. The healthy --pr --auto path.
//   - "stranded" — a PR is open but NO auto-merge is queued: the fixer's
//     `dacli pr --auto` failed to queue it (repo has "Allow auto-merge" off, or
//     GitHub was unreachable) and reported that non-zero. It will NOT self-land,
//     so the loop must not keep counting it as still-landing forever (task 290).
//   - "awaiting-pr" — the branch has not landed and no PR exists yet.
//   - "orphaned" — a PR closed without merging: safe to rebuild from trunk.
//   - "unknown"  — gh and a trunk fetch both failed to answer.
//
// This mirrors features/vcs's checkLanded (gh state first, a fresh-fetch
// ancestor check only when gh finds no PR) but is duplicated, not imported —
// same feature-slice isolation reasoning as runGH above. It goes one step
// further than checkLanded (which the operator reads at `dacli pr status`) by
// splitting an open PR into landing vs. stranded: an unattended loop has no
// human to notice a stranded PR sitting open, so it must tell the two apart
// itself.
func (d *driver) prLandStatus(branch string) string {
	noPR := false
	if out, err := runGH(d.w.Root, "pr", "list", "--head", branch, "--state", "all", "--json", "state,autoMergeRequest", "--limit", "1"); err == nil {
		var prs []struct {
			State            string `json:"state"`
			AutoMergeRequest *struct {
				EnabledAt string `json:"enabledAt"`
			} `json:"autoMergeRequest"`
		}
		if jerr := json.Unmarshal([]byte(out), &prs); jerr == nil && len(prs) == 0 {
			noPR = true
		} else if jerr == nil && len(prs) > 0 {
			switch strings.ToUpper(prs[0].State) {
			case "MERGED":
				return "merged"
			case "OPEN":
				if prs[0].AutoMergeRequest != nil {
					return "landing"
				}
				return "stranded"
			case "CLOSED":
				return "orphaned"
			}
		}
	}
	if d.cfg.dryRun {
		return "unknown"
	}
	hasOrigin := d.hasOrigin()
	b := d.trunkBranch
	if b == "" {
		b = "main"
	}
	// Which trunk ref answers "did this land?" — origin's when there is a
	// remote, the LOCAL branch when there is not.
	//
	// A workspace with no origin used to dead-end here: the fetch failed, every
	// branch reported "unknown", pendingAccept never resolved, and so no task
	// EVER closed. `next` then re-picked the same work every cycle, forever —
	// issue #382's first and worst symptom, on a repo that had merged its
	// branches into trunk perfectly well, just locally.
	trunkRef := "origin/" + b
	if hasOrigin {
		if _, err := gitx.RunNetwork(d.w.Root, "fetch", "-q", "origin", "--", b); err != nil {
			return "unknown"
		}
	} else {
		trunkRef = b
		if _, err := d.git("rev-parse", "--verify", "--quiet", "refs/heads/"+b); err != nil {
			// No remote AND no local trunk: there is nothing to have landed
			// into. Saying "unknown" here is honest — and the loop's no-trunk
			// warning (see runCycle) tells the operator why.
			return "unknown"
		}
	}
	// A branch with no commits beyond trunk is trivially an ancestor of it. Two
	// very different situations produce that, and they must not be conflated:
	//
	//   - a spawn that died before committing — the branch never carried work,
	//     and reporting it "merged" force-accepts an empty task (dacli 168);
	//   - a branch whose commits ARE in trunk because it was merged — which is
	//     precisely a landing.
	//
	// Ancestry alone cannot tell them apart after the merge, so compare tips:
	// an unstarted branch still points at the commit trunk was on when it was
	// cut, while a merged branch's tip is a commit that was made ON it and
	// trunk has moved past (to the merge commit).
	if n, err := d.git("rev-list", "--count", trunkRef+".."+branch); err == nil && strings.TrimSpace(n) == "0" {
		branchTip, e1 := d.git("rev-parse", branch)
		trunkTip, e2 := d.git("rev-parse", trunkRef)
		if e1 != nil || e2 != nil || strings.TrimSpace(branchTip) == strings.TrimSpace(trunkTip) {
			if noPR {
				return "awaiting-pr"
			}
			if !hasOrigin {
				return "orphaned"
			}
			return "unknown"
		}
		return "merged"
	}
	ok, err := gitx.IsAncestor(d.w.Root, branch, trunkRef)
	if err != nil {
		return "unknown"
	}
	if ok {
		return "merged"
	}
	if noPR {
		return "awaiting-pr"
	}
	if !hasOrigin {
		return "orphaned"
	}
	return "unknown"
}

// hasOrigin reports whether the repo has an `origin` remote at all. Cheap,
// local, and the difference between "the PR has not merged yet" and "there is
// no GitHub in this picture" — two states the loop used to conflate into
// "unknown", which is the state that never resolves.
func (d *driver) hasOrigin() bool {
	out, err := d.git("remote")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "origin" {
			return true
		}
	}
	return false
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
//
// The answer is always a branch that exists, or nothing. Two ways it used to
// be neither (dacli 211): on a detached HEAD the last resort was
// `rev-parse --abbrev-ref HEAD`, which returns the literal string "HEAD" — so
// trunkMarker went on to count `origin HEAD` and syncTrunk merged whatever
// arbitrary ref that named; and with origin/HEAD unset (the norm in CI and in
// shallow clones) it fell straight through to a local `main`, which on a repo
// whose work lands on origin/master is a branch nothing ever reaches, making
// every progress measurement a measurement of the wrong thing.
//
// Order of preference, most authoritative first: what origin says its default
// is; then a remote-tracking branch, because trunk is where work LANDS and
// that is a property of the remote, not of this checkout; then a local branch,
// which is all an ordinary offline repo has; then the checked-out branch via
// symbolic-ref, which — unlike rev-parse --abbrev-ref — fails on a detached
// HEAD instead of inventing a name. Nothing resolvable returns "", and the
// callers (trunkMarker, syncTrunk, shipArgs) each already degrade honestly on
// an empty trunk rather than guessing.
func (d *driver) resolveTrunkBranch() string {
	// --into wins outright. A sprint integrates a batch of related work onto
	// its own branch and takes ONE pull request to main at the end, instead of
	// one PR per fix; without this the loop always resolved main and refused
	// the moment the checkout was on the sprint branch ("refusing to operate
	// on the wrong branch"), which made the whole sprint workflow unusable
	// (dacli 332). Validated in cmdLoop before the driver is built, so an
	// unknown branch is a usage error rather than a mid-cycle surprise.
	if d.cfg.into != "" {
		return d.cfg.into
	}
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
		if _, err := d.git("rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+b); err == nil {
			return b
		}
	}
	for _, b := range []string{"main", "master"} {
		if _, err := d.git("rev-parse", "--verify", "--quiet", "refs/heads/"+b); err == nil {
			return b
		}
	}
	// symbolic-ref reports the branch HEAD points AT — it errors on a detached
	// HEAD (and still answers on an unborn branch in a fresh repo), which is
	// exactly the distinction rev-parse --abbrev-ref throws away.
	if out, err := d.git("symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
		if s := strings.TrimSpace(out); s != "" && s != "HEAD" {
			return s
		}
	}
	return ""
}

// trunkMarker is a monotonic count of commits that have reached trunk — local
// OR origin — so it captures both in-cycle local integrations and the async
// GitHub auto-merges the default --pr --auto path produces. It refreshes the
// remote-tracking ref first (so async auto-merges become visible) and degrades
// to the local count when there is no remote.
//
// The bool is the whole point: it reports whether the count could be MEASURED
// at all. Returning a bare 0 when every rev-list variant failed — an index
// lock, a timeout, git briefly unavailable — was indistinguishable from a
// genuinely empty trunk, and the consequences compounded: that cycle computed
// `landed = 0 - prevTrunk`, clamped it to 0 and bumped the thrash streak
// toward a false halt, then set prevTrunk = 0, so the NEXT cycle read the whole
// repository history as this cycle's progress and reset the streak. The thrash
// guard's input must never be a fabricated number (dacli 212).
func (d *driver) trunkMarker() (int, bool) {
	b := d.trunkBranch
	if b == "" {
		b = "main"
	}
	if !d.cfg.dryRun {
		// Network-bound: a hung fetch (wedged network, a credential prompt) must
		// not block the loop — it gets the longer network leash and, on timeout,
		// this degrades to the local-only rev-list count below, the existing
		// best-effort fallback.
		_, _ = gitx.RunNetwork(d.w.Root, "fetch", "-q", "origin", "--", b)
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
				return n, true
			}
		}
	}
	return 0, false
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
		d.logf("  note: local %s not fast-forwarded to origin: %s", b, clikit.FirstLine(out))
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
	if d.cfg.landingExplicit {
		mode := d.cfg.landing.Mode
		if mode == "" {
			mode = model.LandingLocal
		}
		args = append(args, "--landing-mode", string(mode))
		if d.trunkBranch != "" {
			args = append(args, "--landing-base", d.trunkBranch)
		}
		return args
	}
	// A configured base is resolved again by ship from the same project. Only
	// forward the repository-derived fallback, which ship cannot otherwise
	// infer consistently on renamed trunks (dacli 174).
	if d.cfg.landing.Base == "" && d.trunkBranch != "" {
		args = append(args, "--into", d.trunkBranch)
	}
	return args
}

func (d *driver) trunkBase() string {
	if d.trunkBranch != "" {
		return d.trunkBranch
	}
	return d.cfg.landing.Base
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
func (d *driver) reviewPhase(wave ...*store.Task) {
	ref, err := d.ensureImproveTask()
	if err != nil {
		d.logf("  review: could not seed the improvement task: %v", err)
		return
	}
	if len(wave) > 0 && !d.cfg.dryRun {
		if err := d.attachWaveReviewBrief(ref, wave); err != nil {
			d.logf("  review: could not attach the just-completed wave brief: %v", err)
			return
		}
	}
	d.logf("  review: %s audits and files the next improvement…", d.cfg.reviewRole)
	spawn := []string{"spawn", "--task", ref, "--role", d.cfg.reviewRole}
	if d.cfg.perCycleTok > 0 {
		spawn = append(spawn, "--max-tokens", fmt.Sprint(d.cfg.perCycleTok))
	}
	if anchor, findErr := store.FindTask(d.w, ref); findErr == nil {
		spawn = append(spawn, "--timeout", fmt.Sprint(d.workerTimeout(anchor)))
	} else {
		spawn = append(spawn, "--timeout", fmt.Sprint(d.workerTimeout(nil)))
	}
	var result commandresult.Spawn
	var out string
	var runErr error
	if rr, ok := d.run.(resultRunner); ok {
		out, runErr = rr.runResult("review", &result, spawn...)
	} else {
		out, runErr = d.run.run("review", spawn...)
	}
	if runErr != nil {
		d.logf("    %s", reviewFailure(out, runErr, result))
		d.logf("    review produced NO new work this cycle — the backlog grows only here")
		return
	}

	// The review phase's whole output is NEW TASKS, and the next cycle spawns
	// an implementer straight onto them. lint is the check that catches a
	// vague acceptance criterion before tokens are spent proving it was vague
	// — which is precisely what it was written for — and the loop had never
	// run it, so the unattended path was the one place its own quality gate
	// did not apply.
	d.lintFiledWork()
}

// attachWaveReviewBrief makes queued work visible to a reviewer whose checkout
// still points at trunk. Cycle 95 re-filed the wave's own fix because the
// branch had not landed yet; task status alone could not show the commit or
// linked issue that proved it was the same work (issue #668).
func (d *driver) attachWaveReviewBrief(anchorRef string, wave []*store.Task) error {
	anchor, err := store.FindTask(d.w, anchorRef)
	if err != nil {
		return err
	}
	_, base, _ := d.anchorCharter()
	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\nJust-completed wave (treat this as queued work when checking duplicates):\n")
	for _, prior := range wave {
		current, findErr := store.FindTask(d.w, prior.ID)
		if findErr != nil {
			current = prior
		}
		branch := taskBranch(current)
		commit := "none"
		if tip, gitErr := d.git("rev-parse", "--verify", "--quiet", branch); gitErr == nil && strings.TrimSpace(tip) != "" {
			commit = strings.TrimSpace(tip)
		}
		pending := false
		for _, pendingBranch := range d.pendingLand {
			if pendingBranch == branch {
				pending = true
				break
			}
		}
		fmt.Fprintf(&b, "- task %s (%03d-%s); status=%s; branch=%s; commit=%s; linked_issue=%s; pending_pr_landing=%t\n",
			current.ID, current.Seq, current.Slug, current.Status, branch, commit, linkedIssue(current), pending)
	}
	anchor.Doc.SetSection("Context", b.String())
	return store.SaveTask(anchor)
}

func linkedIssue(t *store.Task) string {
	block, ok := t.Doc.Front.GetBlock("github")
	if !ok {
		return "none"
	}
	for _, line := range strings.Split(block, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), ":")
		if found && key == "issue" && strings.TrimSpace(value) != "" {
			return "#" + strings.TrimSpace(value)
		}
	}
	return "none"
}

// lintFiledWork reports ambiguity in the tasks the review phase just filed.
//
// Reported, never fatal. lint flags language, and language it flags is
// sometimes correct — refusing a cycle over a "should" would trade a wave of
// real work for a wording argument. What must not happen is an implementer
// discovering the ambiguity by burning a run on it.
func (d *driver) lintFiledWork() {
	out, err := d.run.run("lint", "lint", "--project", d.cfg.project)
	body := strings.TrimSpace(out)
	if body == "" {
		return
	}
	// lint exits non-zero when it finds something, which is the case worth
	// surfacing rather than swallowing as a failed phase.
	major := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "major") {
			major++
		}
	}
	if err == nil && major == 0 {
		return
	}
	d.logf("  lint: %d major ambiguity finding(s) in the backlog — an implementer spawned onto one of these will discover it the expensive way", major)
	for i, line := range strings.Split(body, "\n") {
		if i >= 5 {
			d.logf("    … `dacli lint --project %s` for the rest", d.cfg.project)
			break
		}
		if strings.TrimSpace(line) != "" {
			d.logf("    %s", line)
		}
	}
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
				return t.ID, nil
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
		Estimate: "1,2,3",
	})
	if err != nil {
		return "", err
	}
	return t.ID, nil
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
		fmt.Sprintf("Standing anchor for the autonomous review phase. Survey the code, tests, CI, and open findings; identify the ONE highest-value improvement grounded in evidence (a failing test, a reviewer finding, a real defect). Before filing, run `dacli task list --project %s --status open` and `dacli task list --project %s --status active` to check whether the backlog already queues it — a prior cycle may have filed the same issue under different wording. `dacli task add` refuses (exit 3) a title that scores as a near-duplicate of existing work, so pick real, distinct scope rather than re-filing and re-running with --force. If the audit finds no distinct task after those duplicate checks, that is an honest result: record a finding naming what you audited and the open/active work that already covers it, then finish this anchor without filing placeholder work. Otherwise file the distinct task with concrete acceptance criteria. Do NOT implement anything here, and do NOT invent speculative work.", d.cfg.project, d.cfg.project),
		[]string{
			"Evidenced exactly one outcome: filed a distinct task grounded in an observed defect, finding, or failing check; or recorded a reviewer finding that the audit found no distinct task after checking open and active work for duplicates",
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

// readyFrontier evaluates the ONE readiness predicate (store.ReadyFrontier)
// over the project. The rule used to be reimplemented here, with a comment
// claiming it mirrored `dacli next` while differing from it on dep types, ref
// resolution, unresolvable refs and candidate status — so the loop could
// silently refuse work `next` was recommending (dacli 240). The rule and its
// four judgement calls now live in store; this is just the workspace read.
func readyFrontier(w *workspace.Workspace, project string) (store.Frontier, error) {
	tasks, err := store.ListTasks(w, project, "")
	if err != nil {
		return store.Frontier{}, err
	}
	return store.ReadyFrontier(tasks), nil
}

// readyTasks returns the workable frontier the loop draws from.
func readyTasks(w *workspace.Workspace, project string) ([]*store.Task, error) {
	fr, err := readyFrontier(w, project)
	if err != nil {
		return nil, err
	}
	return fr.Ready, nil
}

// readyTasks (driver method) is the frontier the BUILD phase draws from, and
// the one place that REPORTS the data faults holding tasks back. A dependency
// ref naming no task blocks its task forever; the loop refusing to run it is
// correct, but refusing in silence is what made 240 a mystery instead of a
// one-line fix. Called once per cycle at the decision point — the frontier
// re-reads elsewhere in the loop stay quiet so the note appears once.
func (d *driver) readyTasks() ([]*store.Task, error) {
	fr, err := readyFrontier(d.w, d.cfg.project)
	if err != nil {
		return nil, err
	}
	for _, line := range fr.ProblemLines() {
		d.logf("  note: %s — fix the ref; this task cannot be scheduled until it resolves", line)
	}
	return fr.Ready, nil
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
		// Exclude the loop anchor, exactly as cmdNext does (insight.go:168).
		// It is a standing review-phase prompt, never implementer work — and
		// it is created UNSIZED (ensureImproveTask passes no Estimate) and
		// never sized, because sizeUnestimated only sizes the wave batch and
		// readiness filters anchors out of that. Including it here meant
		// t.Estimate() failed on it every cycle, so haveCPM went false and the
		// BUILD phase silently fell back to MoSCoW+seq while `dacli next`
		// showed the operator critical-path order. The two are documented to
		// agree; they did not.
		if t.Status != model.StatusDone && t.Status != model.StatusBlocked && !t.IsLoopAnchor() {
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

// resolveLoopLanding uses the same entity-layer precedence as integrate/ship,
// while a journaled policy keeps one bounded run stable across process-level
// checkpoints. Feature isolation prevents importing either landing slice.
func resolveLoopLanding(w *workspace.Workspace, project string, f *clikit.Flags, journal cycleJournal) (model.LandingPolicy, bool, error) {
	p, err := store.LoadProject(w, project)
	if err != nil {
		return model.LandingPolicy{}, false, err
	}
	configured := p.Landing
	if journal.Landing.Mode != "" {
		configured = journal.Landing
	}
	var override model.LandingOverride
	if f.Bool("pr") && f.Bool("no-pr") {
		return model.LandingPolicy{}, false, clikit.Usagef("use either --pr or --no-pr, not both")
	}
	if f.Bool("pr") || f.Bool("no-pr") {
		mode := model.LandingPR
		if f.Bool("no-pr") {
			mode = model.LandingLocal
		}
		override.Mode = &mode
	}
	if len(f.All("into")) > 0 {
		base := f.Get("into")
		override.Base = &base
	}
	effective, explicit, err := model.ResolveLanding(configured, override)
	if err != nil {
		return model.LandingPolicy{}, explicit, clikit.Usagef("%v", err)
	}
	if override.Mode == nil && override.Base == nil && journal.Landing.Mode != "" {
		explicit = journal.LandingExplicit
	}
	return effective, explicit, nil
}

// hasOriginRemote is the package-level twin of driver.hasOrigin, for the
// config path that runs before a driver exists.
func hasOriginRemote(root string) bool {
	out, err := gitx.Run(root, "remote")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "origin" {
			return true
		}
	}
	return false
}

// reportWorkspaceHealth runs doctor's checks once per cycle and surfaces
// anything it finds. It shells to the same binary every other phase uses, so
// the loop sees exactly what an operator running `dacli doctor` would.
func (d *driver) reportWorkspaceHealth() {
	out, err := d.run.run("doctor", "doctor")
	if err != nil {
		// doctor exits non-zero when it FINDS something, which is the case we
		// care about — surface its report rather than swallowing it as a
		// failed phase.
		if s := strings.TrimSpace(out); s != "" {
			d.logf("  workspace health:")
			for _, line := range strings.Split(s, "\n") {
				d.logf("    %s", line)
			}
		}
		return
	}
	if s := strings.TrimSpace(out); s != "" && !strings.Contains(s, "no anti-patterns") {
		d.logf("  workspace health: %s", clikit.FirstLine(s))
	}
}

// sizeUnestimated spawns the roster's estimator on any task in the batch that
// carries no three-point estimate, and says so when it cannot.
//
// This is the gap both wrong halves of the loop audit were groping at. The
// loop already routes by capacity and already ranks by critical path — but
// both read t.Estimate(), and both fall back silently when it is missing: the
// task lands on the fallback role regardless of size, and the whole wave
// reverts to MoSCoW order while still printing as if the critical path were
// in play. Sizing is the input those features were built to consume.
//
// It uses the roster's own estimator role rather than inventing numbers here:
// a three-point estimate is a judgment about the codebase, which is exactly
// what that role exists to make. With no estimator in the roster there is
// nothing honest to do but name the degradation.
func (d *driver) sizeUnestimated(batch []*store.Task) {
	var unsized []*store.Task
	for _, t := range batch {
		if _, ok := t.Estimate(); !ok {
			unsized = append(unsized, t)
		}
	}
	if len(unsized) == 0 {
		return
	}
	if _, ok := store.LoadRole(d.w, estimatorRole); !ok {
		d.logf("  %d task(s) have no estimate — capacity routing and critical-path order both degrade; add an `%s` role, or size them with `dacli task estimate <ref> --estimate o,m,p`",
			len(unsized), estimatorRole)
		return
	}
	for _, t := range unsized {
		ref := t.ID
		d.logf("  sizing %s with the %s role (unsized tasks lose routing and critical-path order)", ref, estimatorRole)
		if out, err := d.run.run("estimate", "spawn", "--task", ref, "--role", estimatorRole); err != nil {
			// Never fatal: an unsized task still builds, just on the fallback
			// role and in MoSCoW order — the pre-existing behavior.
			d.logf("    could not size %s: %s", ref, clikit.FirstLine(out))
		}
	}
}

// estimatorRole is the roster role that sizes an open task. Named once so the
// lookup and the advice can never disagree about what to add.
const estimatorRole = "estimator"

// dropRemoteBranch deletes a task branch from origin after its PR closed
// unmerged.
//
// Left behind, the ref is indistinguishable from work in flight to anything
// that only asks whether it exists — which is exactly what stillPending used
// to do, holding the record push indefinitely on the evidence of a branch
// nobody was going to merge. Deleting it is safe precisely because the PR is
// closed: the work is either abandoned or already elsewhere.
//
// Best-effort. A protected branch, a revoked token or an offline remote must
// never fail a cycle over cleanup; the local clear above is what makes the
// retry fresh, and this only stops the stale ref from misleading a later read.
func (d *driver) dropRemoteBranch(branch string) {
	if !d.hasOrigin() || d.cfg.dryRun {
		return
	}
	if out, err := gitx.RunNetwork(d.w.Root, "push", "origin", "--delete", "--", branch); err != nil {
		d.logf("    could not delete origin/%s (left in place): %s", branch, clikit.FirstLine(out))
	}
}

// reviewFailure turns a failed review spawn into a line that names what
// ACTUALLY went wrong.
//
// The original reported `spawn refused/failed: <FirstLine(out)>`, and the
// first line of a spawn's output is its SUCCESS banner — so a review agent
// that ran for five minutes and was killed on a timeout was reported as
//
//	spawn refused/failed: spawning a-go-auditor-yn0a9b on cc for 303-… (run 01KZPAKYFN)
//
// which names the spawn that worked and says nothing about the failure. A
// policy refusal and a timeout kill were indistinguishable from the loop's
// output, and only `dacli runs list` showed the truth: "outcome: stalled ·
// exit: signal: killed · elapsed: 5m0.022s" (dacli 333).
//
// Deliberately NOT reported: a derived "timeout" figure. The loop knows the
// token budget it passed, not the wall-clock limit the runtime applied, and
// printing the former labelled as the latter trades one misleading message
// for another. The run id is printed instead, because the run record is the
// thing that actually knows.
func reviewFailure(out string, err error, result commandresult.Spawn) string {
	if result.RunID != "" {
		where := fmt.Sprintf(" — `dacli runs list` and `dacli logs %s` for the outcome", clikit.Short(result.RunID, 10))
		// It ran and did not finish. That is a different operator problem from
		// a spawn the gate never let through, and the error — not the banner —
		// is the half that says which.
		return fmt.Sprintf("review spawn started but did not complete: %v%s", err, where)
	}
	// No typed run result: the spawn never got far enough to mint a run, so the
	// output is the refusal and is the more actionable of the two.
	if msg := clikit.FirstLine(out); msg != "" {
		return fmt.Sprintf("review spawn refused: %s", msg)
	}
	return fmt.Sprintf("review spawn failed: %v", err)
}

// reportStillUnsized names the structural mismatch that the thrash guard's
// "no net progress" hides: a task nothing sized, about to meet a role that
// refuses unsized work.
//
// The reporter watched fourteen consecutive no-progress cycles before the
// guard tripped, and had to notice a per-spawn refusal buried above the rollup
// to learn that every task in the backlog was unspawnable. That is one line's
// worth of difference between a five-minute fix and reading a redirected log
// (issue #430).
func (d *driver) reportStillUnsized(batch []*store.Task) {
	var stuck []string
	for _, t := range batch {
		fresh, err := store.FindTask(d.w, fmt.Sprintf("%03d", t.Seq))
		if err != nil {
			continue
		}
		if _, ok := fresh.Estimate(); !ok {
			stuck = append(stuck, fmt.Sprintf("%03d", fresh.Seq))
		}
	}
	if len(stuck) == 0 {
		return
	}
	d.logf("  %d task(s) are STILL unsized after the sizing step (%s) — every capacity-capped role refuses unsized work, so this cycle cannot build them. Size them by hand (`dacli task estimate <ref> --estimate o,m,p --force`) or add an `%s` role.",
		len(stuck), strings.Join(stuck, ", "), estimatorRole)
}
