package execution

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// runID builds a deterministic 26-character id with the same length and
// lexical-ordering properties as a real ULID, so the ordering the retention
// policy depends on is unambiguous in the test itself.
func runID(i int) string { return fmt.Sprintf("01RUN%04dZZZZZZZZZZZZZZZZZ", i) }

// mkRun creates a run directory with an outcome line.
func mkRun(t *testing.T, w *workspace.Workspace, id, outcome string) string {
	t.Helper()
	dir := w.RunDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if outcome != "" {
		if err := os.WriteFile(filepath.Join(dir, "outcome.md"), []byte(outcome), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// `runs prune` is the only thing bounding transcript growth. It must delete the
// OLDEST runs (ULIDs sort lexically by time) and keep exactly --keep of them —
// an off-by-one here either lets the disk fill or silently eats a run an
// operator still needed.
func TestRunsPruneKeepsTheNewestN(t *testing.T) {
	cases := []struct {
		name     string
		total    int
		args     []string
		wantKept int
	}{
		{"default retention is 20", 25, nil, 20},
		{"--keep N", 25, []string{"--keep", "5"}, 5},
		{"keep more than exist prunes nothing", 3, []string{"--keep", "10"}, 3},
		{"keep exactly the count prunes nothing", 7, []string{"--keep", "7"}, 7},
		{"a non-positive --keep falls back to the default", 25, []string{"--keep", "0"}, 20},
		{"keep 1", 25, []string{"--keep", "1"}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newExecWS(t)
			var ids []string
			for i := 0; i < tc.total; i++ {
				id := runID(i) // lexical order == chronological order
				ids = append(ids, id)
				mkRun(t, w, id, "outcome: ok\n")
			}
			ctx, out, _ := newCtx(w.Root)
			if err := cmdRunsPrune(ctx, tc.args); err != nil {
				t.Fatal(err)
			}
			entries, _ := os.ReadDir(w.RunsDir())
			if len(entries) != tc.wantKept {
				t.Fatalf("kept %d run(s), want %d", len(entries), tc.wantKept)
			}
			// The survivors must be the NEWEST ones, not an arbitrary subset.
			kept := map[string]bool{}
			for _, e := range entries {
				kept[e.Name()] = true
			}
			for _, id := range ids[tc.total-tc.wantKept:] {
				if !kept[id] {
					t.Errorf("pruned a run that should have been kept: %s", id)
				}
			}
			for _, id := range ids[:tc.total-tc.wantKept] {
				if kept[id] {
					t.Errorf("kept a run that should have been pruned: %s", id)
				}
			}
			if want := fmt.Sprintf("pruned %d run(s), kept %d", tc.total-tc.wantKept, tc.wantKept); !strings.Contains(out.String(), want) {
				t.Errorf("report %q does not say %q", out, want)
			}
		})
	}
}

// mkLiveRun creates a run at a CHOSEN id whose proc.txt names THIS test
// process — a genuinely live pid with a matching start time, so every liveness
// probe sees a running agent. Nothing is spawned and nothing is ever signalled.
func mkLiveRun(t *testing.T, w *workspace.Workspace, id string) {
	t.Helper()
	pid := os.Getpid()
	start, ok := procmon.ProcStart(pid)
	if !ok {
		t.Skip("ps cannot read this process's start time")
	}
	dir := mkRun(t, w, id, "")
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{
		RunID: id, Child: "a-live", Task: "t-1", Role: "junior", Runtime: "rt",
		PID: pid, PGID: pid, PIDStart: start, Started: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
}

// dacli-208: retention must never delete the state of a run that is STILL
// EXECUTING. proc.txt, the transcript and usage are the only handles dacli has
// on a live agent — remove them and `agents`, `wait` and `kill` all go blind,
// orphaning a process that keeps burning tokens with nobody able to stop it.
// A live run is skipped and SAID OUT LOUD, not silently kept.
func TestRunsPruneNeverDeletesALiveRun(t *testing.T) {
	w := newExecWS(t)
	for i := 0; i < 6; i++ {
		mkRun(t, w, runID(i), "outcome: ok\n")
	}
	// runID(1) is among the oldest — squarely inside the prune window.
	mkLiveRun(t, w, runID(1))

	ctx, out, _ := newCtx(w.Root)
	if err := cmdRunsPrune(ctx, []string{"--keep", "3"}); err != nil {
		t.Fatal(err)
	}

	// The live run and the newest three survive; only the dead old ones go.
	for _, id := range []string{runID(1), runID(3), runID(4), runID(5)} {
		if _, err := os.Stat(w.RunDir(id)); err != nil {
			t.Errorf("run %s should have survived: %v", id, err)
		}
	}
	for _, id := range []string{runID(0), runID(2)} {
		if _, err := os.Stat(w.RunDir(id)); err == nil {
			t.Errorf("dead run %s should have been pruned", id)
		}
	}
	// The live agent's handles specifically — deleting these is the whole bug.
	if _, err := os.Stat(filepath.Join(w.RunDir(runID(1)), "proc.txt")); err != nil {
		t.Errorf("the live run's proc.txt was deleted — the agent is now unobservable: %v", err)
	}

	if !strings.Contains(out.String(), "pruned 2 run(s), kept 4") {
		t.Errorf("report must count the run it refused to delete:\n%s", out)
	}
	if !strings.Contains(out.String(), "still live") || !strings.Contains(out.String(), "a-live") {
		t.Errorf("a skipped live run must be reported with a reason, not silently kept:\n%s", out)
	}
}

// The liveness guard must not become a blanket "keep everything": a run whose
// recorded process is long gone is a ghost and must still be pruned, or the
// retention bound stops holding the moment one stale proc.txt is on disk.
func TestRunsPruneStillDeletesGhostRuns(t *testing.T) {
	w := newExecWS(t)
	for i := 0; i < 4; i++ {
		dir := mkRun(t, w, runID(i), "outcome: ok\n")
		if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{
			RunID: runID(i), Child: "a-ghost", PID: 1 << 30, PGID: 1 << 30, Started: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	ctx, out, _ := newCtx(w.Root)
	if err := cmdRunsPrune(ctx, []string{"--keep", "1"}); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(w.RunsDir()); len(entries) != 1 {
		t.Fatalf("kept %d run(s), want 1 — a dead pid must not block pruning", len(entries))
	}
	if strings.Contains(out.String(), "still live") {
		t.Errorf("a ghost record was reported as live:\n%s", out)
	}
}

// `runs list` is a read-only command over a directory it does not control, so
// it must never crash on what it finds there. It shortens each run id to ten
// characters for display; an unguarded n[:10] slice panics the whole command on
// any run dir whose name is shorter — which every sibling reader (cmdAgents,
// cmdWait, cmdSpawn's logging) already guards against with a length-capped
// shortener. This is the regression test for that guard.
func TestRunsListToleratesAShortRunDirName(t *testing.T) {
	w := newExecWS(t)
	mkRun(t, w, "short", "outcome: ok\n")
	mkRun(t, w, runID(1), "outcome: failed\n")

	var out *bytes.Buffer
	panicked := func() (p bool) {
		defer func() { p = recover() != nil }()
		var ctx *clikit.Ctx
		ctx, out, _ = newCtx(w.Root)
		if err := cmdRunsList(ctx, nil); err != nil {
			t.Errorf("cmdRunsList: %v", err)
		}
		return
	}()
	if panicked {
		t.Fatal("cmdRunsList panicked on a run-dir name shorter than the display width")
	}
	if !strings.Contains(out.String(), "short") {
		t.Errorf("the short-named run was not listed:\n%s", out)
	}
	if !strings.Contains(out.String(), runID(1)[:10]) {
		t.Errorf("the normal-length run was not listed:\n%s", out)
	}
}

func TestRunsPruneRejectsUnknownFlags(t *testing.T) {
	w := newExecWS(t)
	ctx, _, _ := newCtx(w.Root)
	err := cmdRunsPrune(ctx, []string{"--keepp", "5"})
	if code := clikit.ExitCode(err); code != 2 {
		t.Fatalf("exit %d, want 2 (a typo'd flag must not silently prune the default)", code)
	}
}

// `runs list` shows newest first (ULIDs reverse-sorted) and flattens each
// outcome onto one line. A run with no outcome recorded says so rather than
// being omitted — an invisible run is an unauditable one.
func TestRunsListNewestFirst(t *testing.T) {
	w := newExecWS(t)
	mkRun(t, w, runID(1), "outcome: ok\nexit: 0\n")
	mkRun(t, w, runID(2), "outcome: failed\nexit: 1\n")
	mkRun(t, w, runID(3), "")

	ctx, out, _ := newCtx(w.Root)
	if err := cmdRunsList(ctx, nil); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], runID(3)[:10]) {
		t.Errorf("newest run must be first; got %q", lines[0])
	}
	if !strings.Contains(lines[0], "no outcome recorded") {
		t.Errorf("a run with no outcome must still be listed and labelled: %q", lines[0])
	}
	if !strings.HasPrefix(lines[2], runID(1)[:10]) {
		t.Errorf("oldest run must be last; got %q", lines[2])
	}
	if !strings.Contains(lines[1], "outcome: failed · exit: 1") {
		t.Errorf("multi-line outcome must flatten with ' · '; got %q", lines[1])
	}
}

// `runs show` resolves a run-id PREFIX (nobody types a whole ULID) and prints
// the four records in a fixed order: what it was told, what happened, the
// frozen brief, the transcript.
func TestRunsShowByPrefix(t *testing.T) {
	w := newExecWS(t)
	dir := mkRun(t, w, "01RUN0001", "outcome: ok\n")
	for name, body := range map[string]string{
		"invocation.txt": "run: 01RUN0001\nchild: a-1\n",
		"brief.md":       "## Task: do the thing\n",
		"transcript.log": "child said hello\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdRunsShow(ctx, []string{"01RUN"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	order := []string{"=== invocation.txt ===", "=== outcome.md ===", "=== brief.md ===", "=== transcript.log ==="}
	at := -1
	for _, want := range order {
		i := strings.Index(got, want)
		if i < 0 {
			t.Fatalf("missing section %q in:\n%s", want, got)
		}
		if i < at {
			t.Errorf("section %q out of order", want)
		}
		at = i
	}
	if !strings.Contains(got, "do the thing") || !strings.Contains(got, "child said hello") {
		t.Errorf("run record bodies not printed:\n%s", got)
	}

	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRunsShow(ctx2, []string{"01NOPE"})); code != 4 {
		t.Errorf("unknown run prefix exit %d, want 4 (not found)", code)
	}
	ctx3, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRunsShow(ctx3, nil)); code != 2 {
		t.Errorf("missing argument exit %d, want 2", code)
	}
}

// liveAgents must be runaways-included, ghosts-excluded: liveness is PROBED,
// never trusted from the file. A run whose recorded pid is long gone must not
// resurface in `agents` / `kill` / `wait`.
func TestLiveAgentsProbesLivenessAndExcludesGhosts(t *testing.T) {
	w := newExecWS(t)

	// A ghost: a plausible-looking record naming a pid that cannot be alive.
	dir := mkRun(t, w, "01RUN0001", "")
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{
		RunID: "01RUN0001", Child: "a-ghost", PID: 1 << 30, PGID: 1 << 30, Started: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	// A run with no proc.txt at all is simply skipped, not an error.
	mkRun(t, w, "01RUN0002", "outcome: ok\n")

	if got := liveAgents(w); len(got) != 0 {
		t.Fatalf("ghost record resurfaced as live: %+v", got)
	}

	live := writeLiveProcRecord(t, w, nil)
	got := liveAgents(w)
	if len(got) != 1 || got[0].RunID != live.RunID {
		t.Fatalf("liveAgents = %+v, want just the live record %s", got, live.RunID)
	}

	// readProcByRef finds a run by id-prefix OR by child id — and finds the
	// GHOST too, because `logs` must work on a finished run.
	if rec, ok := readProcByRef(w, "01RUN0001"); !ok || rec.Child != "a-ghost" {
		t.Errorf("readProcByRef by run prefix = (%+v, %v)", rec, ok)
	}
	if rec, ok := readProcByRef(w, "a-ghost"); !ok || rec.RunID != "01RUN0001" {
		t.Errorf("readProcByRef by child id = (%+v, %v)", rec, ok)
	}
	if _, ok := readProcByRef(w, "no-such-ref"); ok {
		t.Error("readProcByRef matched a ref that does not exist")
	}
}

// `dacli wait` must not call a run finished while forked children are still
// running under its group — landing mid-commit is exactly the failure
// runStillLive exists to prevent. So it is the OR of leader-alive and
// group-alive, not leader-alive alone.
func TestRunStillLive(t *testing.T) {
	dead := procmon.Record{PID: 1 << 30, PGID: 1 << 30}
	if runStillLive(dead) {
		t.Error("a record with a dead leader and a dead group must not be live")
	}
	if runStillLive(procmon.Record{PID: 0, PGID: 0}) {
		t.Error("a zero record must not be live")
	}
	// This process is alive and is its own group member, so both arms hold.
	pid := os.Getpid()
	start, ok := procmon.ProcStart(pid)
	if !ok {
		t.Skip("ps cannot read this process's start time")
	}
	if !runStillLive(procmon.Record{PID: pid, PGID: pid, PIDStart: start}) {
		t.Error("a live leader must count as live")
	}
	// The leader-dead-but-group-alive arm needs a real process group; it lives
	// in runstilllive_unix_test.go.
}

// finalizeRun derives a detached run's outcome from what the child actually
// wrote to the workspace — there is no exit code to read, so effects are the
// only honest signal. It also harvests the usage the detached path could not
// capture live.
func TestFinalizeRunDerivesOutcomeFromEffects(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "detached task", store.TaskOpts{Accept: []string{"box one", "box two"}})

	dir := mkRun(t, w, "01RUN0001", "outcome: running (detached)\n")
	if err := os.WriteFile(filepath.Join(dir, "transcript.log"), []byte(streamJSONFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := eventlog.Append(w, "a-child", model.EventFinding, task.ID, "", "found something"); err != nil {
		t.Fatal(err)
	}

	rec := procmon.Record{RunID: "01RUN0001", Child: "a-child", Task: task.ID, Started: time.Now()}
	summary := finalizeRun(w, rec)

	if !strings.Contains(summary, "a-child: done") {
		t.Errorf("summary = %q, want a done outcome (the child left events behind)", summary)
	}
	if !strings.Contains(summary, "1 event(s)") || !strings.Contains(summary, "acceptance 0/2") {
		t.Errorf("summary = %q, want the effect counts", summary)
	}
	outcome, err := os.ReadFile(filepath.Join(dir, "outcome.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(outcome), "running (detached)") {
		t.Errorf("the placeholder outcome was not overwritten:\n%s", outcome)
	}
	// The detached path never saw a live stream, so usage is harvested here.
	usage, err := os.ReadFile(filepath.Join(dir, "usage.txt"))
	if err != nil {
		t.Fatalf("usage was not harvested from the stream-json transcript: %v", err)
	}
	for _, want := range []string{"output_tokens: 345", "input_tokens: 1200", "num_turns: 2"} {
		if !strings.Contains(string(usage), want) {
			t.Errorf("usage.txt missing %q:\n%s", want, usage)
		}
	}
}

// A child that left NOTHING behind must be reported as such, not as "done" —
// a run with no visible result is the signal that the spawn was wasted.
func TestFinalizeRunReportsNoVisibleResult(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "quiet task", store.TaskOpts{Accept: []string{"box one"}})
	mkRun(t, w, "01RUN0001", "outcome: running (detached)\n")

	got := finalizeRun(w, procmon.Record{RunID: "01RUN0001", Child: "a-quiet", Task: task.ID, Started: time.Now()})
	if !strings.Contains(got, "no visible result") {
		t.Errorf("summary = %q, want 'no visible result'", got)
	}
}

// A text runtime's transcript carries no result event, so nothing must be
// written — text runtimes stay byte-for-byte unaffected by usage harvesting.
func TestFinalizeRunLeavesTextRuntimesAlone(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "text task", store.TaskOpts{})
	dir := mkRun(t, w, "01RUN0001", "")
	if err := os.WriteFile(filepath.Join(dir, "transcript.log"), []byte("plain output\nno json here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	finalizeRun(w, procmon.Record{RunID: "01RUN0001", Child: "a-child", Task: task.ID, Started: time.Now()})
	if _, err := os.Stat(filepath.Join(dir, "usage.txt")); !os.IsNotExist(err) {
		t.Errorf("usage.txt fabricated for a text runtime (err %v)", err)
	}
}

// An already-captured usage.txt (the foreground path wrote it) must not be
// re-derived and clobbered.
func TestFinalizeRunDoesNotClobberCapturedUsage(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "task", store.TaskOpts{})
	dir := mkRun(t, w, "01RUN0001", "")
	if err := os.WriteFile(filepath.Join(dir, "transcript.log"), []byte(streamJSONFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	const captured = "output_tokens: 99\n"
	if err := os.WriteFile(filepath.Join(dir, "usage.txt"), []byte(captured), 0o644); err != nil {
		t.Fatal(err)
	}
	finalizeRun(w, procmon.Record{RunID: "01RUN0001", Child: "a-child", Task: task.ID, Started: time.Now()})
	got, _ := os.ReadFile(filepath.Join(dir, "usage.txt"))
	if string(got) != captured {
		t.Errorf("live-captured usage was clobbered: %q", got)
	}
}

func TestWriteUsageFormat(t *testing.T) {
	dir := t.TempDir()
	writeUsage(dir, streamUsage{InputTokens: 10, OutputTokens: 20, NumTurns: 3, CostUSD: 1.5})
	raw, err := os.ReadFile(filepath.Join(dir, "usage.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := "output_tokens: 20\ninput_tokens: 10\nnum_turns: 3\ncost_usd: 1.500000\n"
	if string(raw) != want {
		t.Errorf("usage.txt = %q, want %q", raw, want)
	}
}

// `dacli agents` reports "no live agents" rather than printing nothing — an
// empty screen is indistinguishable from a broken command.
func TestAgentsReportsEmptyState(t *testing.T) {
	w := newExecWS(t)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdAgents(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "no live agents") {
		t.Errorf("cmdAgents printed %q, want an explicit empty state", out)
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAgents(ctx2, []string{"--reapp"})); code != 2 {
		t.Errorf("a typo'd --reap must be a usage error, not a silent no-op")
	}
}

// task 268: a detached run whose process is gone but which nobody `wait`ed on
// keeps a stale "running (detached)" outcome forever, so a silent child that
// produced nothing is indistinguishable from one still working. `dacli agents`
// must finalize it on observation — deriving the honest outcome from effects —
// and surface it loudly, not only `dacli wait`.
func TestAgentsFinalizesGoneDetachedRuns(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "quiet detached task", store.TaskOpts{Accept: []string{"box one"}})

	// A detached run left in the placeholder state, naming a pid that cannot be
	// alive — the child produced no events and checked no acceptance.
	dir := mkRun(t, w, runID(1), "outcome: running (detached)\nchild: a-quiet\ntask: "+task.ID+"\n")
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{
		RunID: runID(1), Child: "a-quiet", Task: task.ID, PID: 1 << 30, PGID: 1 << 30, Started: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdAgents(ctx, nil); err != nil {
		t.Fatal(err)
	}

	// The stale placeholder must be gone from disk — the record no longer lies.
	outcome, err := os.ReadFile(filepath.Join(dir, "outcome.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(outcome), "running (detached)") {
		t.Errorf("agents left the outcome at 'running' for a dead process:\n%s", outcome)
	}
	if !strings.Contains(string(outcome), "no visible result") {
		t.Errorf("a child that left nothing must be finalized as 'no visible result':\n%s", outcome)
	}
	// And it must be said out loud, so the silent agent is loud at first sight.
	if !strings.Contains(out.String(), "no visible result") || !strings.Contains(out.String(), "a-quiet") {
		t.Errorf("agents must surface the just-finalized dead run, not swallow it:\n%s", out)
	}
}

// The sweep must not fire early: a detached run whose process is STILL live
// keeps its placeholder, or `agents` would report a working agent as finished.
func TestAgentsLeavesLiveDetachedRunAlone(t *testing.T) {
	w := newExecWS(t)
	pid := os.Getpid()
	start, ok := procmon.ProcStart(pid)
	if !ok {
		t.Skip("ps cannot read this process's start time")
	}
	dir := mkRun(t, w, runID(1), "outcome: running (detached)\nchild: a-live\ntask: t-1\n")
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{
		RunID: runID(1), Child: "a-live", Task: "t-1", PID: pid, PGID: pid, PIDStart: start, Started: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdAgents(ctx, nil); err != nil {
		t.Fatal(err)
	}
	outcome, _ := os.ReadFile(filepath.Join(dir, "outcome.md"))
	if !strings.Contains(string(outcome), "running (detached)") {
		t.Errorf("a live detached run was finalized prematurely:\n%s", outcome)
	}
	if strings.Contains(out.String(), "finalized") {
		t.Errorf("agents claimed to finalize a still-live run:\n%s", out)
	}
}

// `dacli wait` with nothing to wait for returns cleanly instead of blocking
// for the full hour-long default timeout.
func TestWaitWithNothingPendingReturnsImmediately(t *testing.T) {
	w := newExecWS(t)
	ctx, out, _ := newCtx(w.Root)
	done := make(chan error, 1)
	go func() { done <- cmdWait(ctx, nil) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cmdWait blocked with nothing pending")
	}
	if !strings.Contains(out.String(), "nothing to wait for") {
		t.Errorf("cmdWait printed %q", out)
	}
}

// `dacli kill` terminates process groups — a privileged, irreversible effect
// whose target pgid comes from a file any rw child can write. It must stay off
// the read-only surface entirely.
func TestKillRequiresRW(t *testing.T) {
	w := newExecWS(t)
	_, token, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, "junior", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(agentid.EnvVar, token)

	ctx, _, _ := newCtx(w.Root)
	err = cmdKill(ctx, []string{"--all"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("cmdKill as a read-only agent: exit %d, want 3 (err %v)", code, err)
	}
	if !strings.Contains(err.Error(), "killing an agent") {
		t.Errorf("refusal %q does not name the action", err)
	}
}

// Defining a runtime names the binary and env every child executes with — the
// most privileged write in the system. A read-only agent that could add one
// could then spawn into an arbitrary binary.
func TestRuntimeAddRequiresRW(t *testing.T) {
	w := newExecWS(t)
	_, token, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, "junior", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(agentid.EnvVar, token)

	ctx, _, _ := newCtx(w.Root)
	err = cmdRuntimeAdd(ctx, []string{"evil", "--binary", "/bin/sh"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("cmdRuntimeAdd as a read-only agent: exit %d, want 3 (err %v)", code, err)
	}
	if _, lerr := store.LoadRuntime(w, "evil"); lerr == nil {
		t.Error("a refused runtime add still wrote the adapter")
	}
}

// The env denylist is enforced at the command boundary, not only in the helper:
// `runtime add --env ANTHROPIC_API_KEY` must be refused outright, or the
// no-inherited-credential rule is one CLI invocation away from being undone.
func TestRuntimeAddRefusesDeniedEnvPassthrough(t *testing.T) {
	w := newExecWS(t)
	ctx, _, _ := newCtx(w.Root)
	err := cmdRuntimeAdd(ctx, []string{"leaky", "--binary", "/bin/sh", "--env", "PATH", "--env", "ANTHROPIC_API_KEY"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("exit %d, want 3 (err %v)", code, err)
	}
	if _, lerr := store.LoadRuntime(w, "leaky"); lerr == nil {
		t.Error("a refused runtime add still wrote the adapter")
	}
}

// A preset is the shipped adapter; --binary and friends override individual
// fields on top of it without discarding the rest.
func TestRuntimeAddPresetAndOverrides(t *testing.T) {
	w := newExecWS(t)
	ctx, _, _ := newCtx(w.Root)
	if err := cmdRuntimeAdd(ctx, []string{"cc", "--preset", "claude-code", "--model-flag", "--model-name"}); err != nil {
		t.Fatal(err)
	}
	rt, err := store.LoadRuntime(w, "cc")
	if err != nil {
		t.Fatal(err)
	}
	if rt.Binary != "claude" || rt.Mode != "arg" || rt.Flag != "-p" {
		t.Errorf("preset fields lost: %+v", rt)
	}
	if rt.ModelFlag != "--model-name" {
		t.Errorf("--model-flag override lost: %q", rt.ModelFlag)
	}
	if rt.UsageFormat != "stream-json" {
		t.Errorf("the claude-code preset must default usage_format to stream-json, got %q", rt.UsageFormat)
	}
	// The shipped preset must never forward a credential.
	if bad := deniedEnvPassthrough(rt.Env); bad != "" {
		t.Errorf("the claude-code preset forwards a denied env %q", bad)
	}

	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRuntimeAdd(ctx2, []string{"x", "--preset", "no-such-preset"})); code != 2 {
		t.Error("an unknown preset must be a usage error")
	}
}

// Every shipped preset is a security surface in itself: none may declare an
// env passthrough on the denylist. Asserted by enumeration so a new preset
// cannot ship with one.
func TestNoPresetForwardsACredential(t *testing.T) {
	for name, rt := range presets {
		if bad := deniedEnvPassthrough(rt.Env); bad != "" {
			t.Errorf("preset %q forwards denied env %q", name, bad)
		}
	}
}
