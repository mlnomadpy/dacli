package execution

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/ulid"
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

// TestRuntimeDoctorSeparatesDeclaredVerifiedAndFailedRO is the regression for
// dacli 365: a sandbox flag in a committed adapter is an assumption, not proof
// that this installed CLI accepts it. The fixtures are local shell scripts;
// the probe must never need a model or a network service.
func TestRuntimeDoctorSeparatesDeclaredVerifiedAndFailedRO(t *testing.T) {
	w := newExecWS(t)
	fixture := func(name string, sandboxExit int) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), name)
		body := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo fixture-1.0; exit 0; fi\nif [ %d -eq 0 ]; then echo '  --allowedTools TOOLS'; fi\nexit %d\n", sandboxExit, sandboxExit)
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		return path
	}
	verifiedBin := fixture("verified", 0)
	unknownBin := fixture("unknown", 0)
	failedBin := fixture("failed", 7)
	silentBin := filepath.Join(t.TempDir(), "silent")
	if err := os.WriteFile(silentBin, []byte("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo fixture-1.0; fi\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustRuntime(t, w, store.Runtime{Name: "verified", Binary: verifiedBin, Mode: "stdin", SandboxRO: []string{"--allowedTools", "Read,Grep"}})
	mustRuntime(t, w, store.Runtime{Name: "declared", Binary: unknownBin, Mode: "stdin", SandboxRO: []string{"--vendor-plan-mode"}})
	mustRuntime(t, w, store.Runtime{Name: "failed", Binary: failedBin, Mode: "stdin", SandboxRO: []string{"--allowedTools", "Read,Grep"}})
	mustRuntime(t, w, store.Runtime{Name: "silent", Binary: silentBin, Mode: "stdin", SandboxRO: []string{"--allowedTools", "Read,Grep"}})

	ctx, out, _ := newCtx(w.Root)
	if err := cmdRuntimeDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"verified", "sandbox verified",
		"declared", "sandbox unknown (declared, not probeable",
		"failed", "sandbox probe failed",
		"silent", "help did not advertise --allowedTools",
		"contract: prompt=declared model=unsupported result=unsupported usage=unsupported timeout=verified cancellation=verified read-only=verified workspace-write=declared exit=verified",
		"contract: prompt=declared model=unsupported result=unsupported usage=unsupported timeout=verified cancellation=verified read-only=declared workspace-write=declared exit=verified",
		"contract: prompt=declared model=unsupported result=unsupported usage=unsupported timeout=verified cancellation=verified read-only=failed workspace-write=declared exit=verified",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("doctor output does not contain %q:\n%s", want, out.String())
		}
	}

	for name, want := range map[string]store.RuntimeROProbe{
		"verified": store.RuntimeROVerified,
		"declared": store.RuntimeROUnknown,
		"failed":   store.RuntimeROFailed,
		"silent":   store.RuntimeROFailed,
	} {
		rt, err := store.LoadRuntime(w, name)
		if err != nil {
			t.Fatal(err)
		}
		path, err := exec.LookPath(rt.Binary)
		if err != nil {
			t.Fatal(err)
		}
		rt = store.HydrateRuntimeROProbe(w, rt, path)
		if rt.ROProbe != want {
			t.Errorf("%s probe = %q, want %q", name, rt.ROProbe, want)
		}
		_, sandboxErr := sandboxFor(ctx, rt, model.GrantRO, false)
		if gotAllowed := sandboxErr == nil; gotAllowed != (want == store.RuntimeROVerified) {
			t.Errorf("%s ro spawn allowed = %v, want %v (err %v)", name, gotAllowed, want == store.RuntimeROVerified, sandboxErr)
		}
	}
}

func TestVerifyLoadsPersistedRuntimeROProbe(t *testing.T) {
	w := newExecWS(t)
	bin := filepath.Join(t.TempDir(), "verified-runtime")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	rt := store.Runtime{Name: "verified", Binary: bin, Mode: "stdin", SandboxRO: []string{"--allowedTools", "Read,Grep"}}
	mustRuntime(t, w, rt)
	if err := store.SaveRuntimeROProbe(w, rt, bin, store.RuntimeROVerified, "verified by runtime doctor"); err != nil {
		t.Fatal(err)
	}

	raw, err := store.LoadRuntime(w, rt.Name)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	if _, err := sandboxFor(ctx, raw, model.GrantRO, false); err == nil {
		t.Fatal("raw declaration unexpectedly passed the read-only safety gate")
	}

	loaded, err := loadVerifyRuntime(w, rt.Name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sandboxFor(ctx, loaded, model.GrantRO, false); err != nil {
		t.Fatalf("verify runtime rejected persisted doctor evidence: %v", err)
	}
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
// the records in a fixed order, including durable optional-artifact warnings.
func TestRunsShowByPrefix(t *testing.T) {
	w := newExecWS(t)
	dir := mkRun(t, w, "01RUN0001", "outcome: ok\n")
	for name, body := range map[string]string{
		"invocation.txt":  "run: 01RUN0001\nchild: a-1\n",
		"brief.md":        "## Task: do the thing\n",
		"transcript.log":  "child said hello\n",
		"diagnostics.txt": "could not record optional usage.txt\n",
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
	order := []string{"=== invocation.txt ===", "=== outcome.md ===", "=== brief.md ===", "=== transcript.log ===", "=== diagnostics.txt ==="}
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
	if !strings.Contains(got, "could not record optional usage.txt") {
		t.Errorf("durable warning not surfaced:\n%s", got)
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
		RunID: "01RUN0001", Child: "a-ghost", PID: 1 << 30, PGID: 1 << 30,
		Started: time.Now().Add(-runStartupGrace - time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	// A run with no proc.txt at all is simply skipped, not an error.
	mkRun(t, w, "01RUN0002", "outcome: ok\n")

	if got, err := liveAgents(w); err != nil || len(got) != 0 {
		t.Fatalf("liveAgents = (%+v, %v), want (empty, nil)", got, err)
	}

	live := writeLiveProcRecord(t, w, nil)
	got, err := liveAgents(w)
	if err != nil || len(got) != 1 || got[0].RunID != live.RunID {
		t.Fatalf("liveAgents = (%+v, %v), want just the live record %s", got, err, live.RunID)
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

// Issue #588: finalizeRun retires the agent and replaces the running
// placeholder before it returns. The transcript was written moments earlier,
// so the grace heuristic used to keep that terminal run's claim live and
// refuse an immediate follow-up while `agents` could otherwise report nobody.
func TestTerminalOutcomeReleasesClaimBeforeTranscriptGraceExpires(t *testing.T) {
	w := newExecWS(t)
	rec := writeLiveProcRecord(t, w, []string{"internal/features/execution"})
	runDir := w.RunDir(rec.RunID)
	if err := os.WriteFile(filepath.Join(runDir, "transcript.log"), []byte("finished\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "outcome.md"), []byte("outcome: done (detached)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if live, reason := runLifecycleLive(w, rec, time.Now()); live || reason != "" {
		t.Fatalf("terminal run remained live through %q", reason)
	}
	plan := &launchPlan{w: w, Claims: []string{"internal/features/execution"}}
	if err := gateClaimOverlap(nil, plan); err != nil {
		t.Fatalf("terminal run retained its path claim: %v", err)
	}

	// The exact running placeholder remains non-terminal; a genuinely live
	// process must still fence overlapping work.
	if err := os.WriteFile(filepath.Join(runDir, "outcome.md"), []byte(detachedRunningPlaceholder+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec.Outcome = ""
	rec.Claims = []string{"internal/features/execution"}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	if err := gateClaimOverlap(nil, plan); err == nil || !strings.Contains(err.Error(), "path-claim conflict") {
		t.Fatalf("live run did not retain its claim: %v", err)
	}
}

// makeRunsDirUnreadable replaces the runs directory with a regular file, so
// os.ReadDir(w.RunsDir()) fails with a non-ENOENT error (ENOTDIR) — the
// transient-fault shape that must not be confused with "no runs yet"
// (mirrors internal/gates' unreadableTasksProject).
func makeRunsDirUnreadable(t *testing.T, w *workspace.Workspace) {
	t.Helper()
	if err := os.RemoveAll(w.RunsDir()); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(w.RunsDir()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(w.RunsDir(), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// dacli 337: liveAgents must tell "no runs yet"
// (the directory does not exist — normal, empty result) apart from "the runs
// directory exists but cannot be read" (a real fault). Before this fix both
// collapsed to an empty result, so a permission or I/O fault silently read as
// "nobody is working" to every caller — including gateClaimOverlap, a
// launch-time gate that must fail closed rather than wave a spawn through
// believing no other agent could be holding an overlapping claim.
func TestLiveAgentsFailsOnUnreadableRunsDir(t *testing.T) {
	w := newExecWS(t)

	// No runs yet: the directory has never been created. Empty, not an error.
	if got, err := liveAgents(w); err != nil || len(got) != 0 {
		t.Fatalf("liveAgents on a nonexistent runs dir = (%+v, %v), want (empty, nil)", got, err)
	}
	makeRunsDirUnreadable(t, w)

	if got, err := liveAgents(w); err == nil {
		t.Fatalf("liveAgents on an unreadable runs dir = (%+v, nil), want a non-nil error", got)
	}
}

// The WIP-facing surface: gateClaimOverlap decides whether a new spawn's
// --claim collides with a currently-live agent's. An unreadable runs
// directory means it cannot rule out a collision, so it must refuse
// (fail closed) rather than pass on an empty liveAgents result.
func TestGateClaimOverlapFailsClosedOnUnreadableRunsDir(t *testing.T) {
	w := newExecWS(t)
	makeRunsDirUnreadable(t, w)

	p := &launchPlan{w: w, Claims: []string{"internal/foo"}}
	err := gateClaimOverlap(nil, p)
	if err == nil {
		t.Fatal("gateClaimOverlap passed on an unreadable runs dir — cannot rule out a claim overlap it never read")
	}
	if clikit.ExitCode(err) == 0 {
		t.Errorf("gateClaimOverlap error %v carries a success exit code", err)
	}
}

// makeAgentsDirUnreadable replaces the agents directory with a regular file,
// so os.ReadDir(w.AgentsDir()) fails with a non-ENOENT error (ENOTDIR) —
// mirrors makeRunsDirUnreadable above, applied to the WIP-count's own
// directory rather than the runs tree.
func makeAgentsDirUnreadable(t *testing.T, w *workspace.Workspace) {
	t.Helper()
	if err := os.RemoveAll(w.AgentsDir()); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(w.AgentsDir()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(w.AgentsDir(), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// dacli 341: gateRoleWIP decides whether a new spawn would push a role over
// its WIP cap. An unreadable agents directory means it cannot rule that out,
// so it must refuse (fail closed) rather than pass on store.ActiveInRole
// silently reading the fault as "zero agents hold the role" — the same
// "a gate must never certify what it could not read" rule
// TestGateClaimOverlapFailsClosedOnUnreadableRunsDir already pins for
// gateClaimOverlap's runs dir (dacli 337).
func TestGateRoleWIPFailsClosedOnUnreadableAgentsDir(t *testing.T) {
	w := newExecWS(t)
	makeAgentsDirUnreadable(t, w)

	p := &launchPlan{w: w, HasRole: true, RoleName: "junior", Role: team.Role{Name: "junior", WIP: 1}}
	err := gateRoleWIP(nil, p)
	if err == nil {
		t.Fatal("gateRoleWIP passed on an unreadable agents dir — cannot rule out the role already being at its WIP cap")
	}
	if clikit.ExitCode(err) == 0 {
		t.Errorf("gateRoleWIP error %v carries a success exit code", err)
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

// `dacli agents` must show whether a live agent is actually moving — RAM/CPU
// and uptime alone can't tell a reasoning agent from a wedged one (dacli 270).
// This drives cmdAgents' real output (not agentstate.Derive directly) to prove
// the printed line carries the state, and that a stalled agent reads visually
// distinct from a busy one without passing --tail.
func TestAgentsPrintsStateWithoutTail(t *testing.T) {
	w := newExecWS(t)
	rec := writeLiveProcRecord(t, w, nil)
	transcript := filepath.Join(w.RunDir(rec.RunID), "transcript.log")

	if err := os.WriteFile(transcript, []byte("Looking at the approach.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, out, _ := newCtx(w.Root)
	if err := cmdAgents(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[thinking]") {
		t.Errorf("cmdAgents = %q, want the thinking state without --tail", out.String())
	}

	// A frozen transcript, old enough to cross the stall window, must print the
	// attention-grabbing uppercase form — visually distinct from the lowercase
	// "healthy" state above, without needing --tail.
	old := time.Now().Add(-5 * time.Minute)
	if err := os.Chtimes(transcript, old, old); err != nil {
		t.Fatal(err)
	}
	ctx2, out2, _ := newCtx(w.Root)
	if err := cmdAgents(ctx2, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2.String(), "[STALLED]") {
		t.Errorf("cmdAgents = %q, want the STALLED state (uppercase, visually distinct)", out2.String())
	}
}

// A live agent whose task has an outstanding `dacli ask` is reported blocked
// — not acting or thinking — sourced from agentstate.Derive, the same
// function the dashboard calls, rather than a second CLI-only guess.
func TestAgentsPrintsBlockedForAnOutstandingAsk(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "needs an answer", store.TaskOpts{})
	if err := store.MoveTask(w, task, model.StatusBlocked); err != nil {
		t.Fatal(err)
	}

	pid := os.Getpid()
	start, ok := procmon.ProcStart(pid)
	if !ok {
		t.Skip("ps cannot read this process's start time")
	}
	runID := ulid.New()
	dir := w.RunDir(runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "transcript.log"), []byte("[tool: Read]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{
		RunID: runID, Child: "a-blocked", Task: task.Slug, Role: "junior", Runtime: "rt",
		PID: pid, PGID: pid, PIDStart: start, Started: time.Now(),
	}
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdAgents(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[BLOCKED]") {
		t.Errorf("cmdAgents = %q, want the BLOCKED state (task has an outstanding ask)", out.String())
	}
}

// A detached run whose process is gone is finalized by the explicit lifecycle
// command, not by a status read. wait derives the outcome from durable effects
// and surfaces a silent child's lack of results.
func TestWaitFinalizesGoneDetachedRuns(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "quiet detached task", store.TaskOpts{Accept: []string{"box one"}})

	// A detached run left in the placeholder state, naming a pid that cannot be
	// alive — the child produced no events and checked no acceptance.
	dir := mkRun(t, w, runID(1), "outcome: running (detached)\nchild: a-quiet\ntask: "+task.ID+"\n")
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{
		RunID: runID(1), Child: "a-quiet", Task: task.ID, PID: 1 << 30, PGID: 1 << 30,
		Started: time.Now().Add(-runStartupGrace - time.Second),
	}); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdWait(ctx, []string{runID(1)}); err != nil {
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
	// And it must be said out loud by the lifecycle command.
	if !strings.Contains(out.String(), "no visible result") || !strings.Contains(out.String(), "a-quiet") {
		t.Errorf("wait must surface the just-finalized dead run, not swallow it:\n%s", out)
	}
}

// Runs 01KZVSF64J and 01KZVSFBXR were finalized 12s and 7s after spawn even
// though both Codex transcripts kept advancing for minutes. Codex's registered
// guardian had disappeared during startup, so process identity alone was a
// false negative. Every lifecycle reader must retain such runs while startup
// grace or durable transcript activity says work is still happening.
func TestWaitKeepsFreshCodexRunsLiveDuringRegistrationStartup(t *testing.T) {
	w := newExecWS(t)
	for i, age := range []time.Duration{12 * time.Second, 7 * time.Second} {
		id := runID(i + 1)
		dir := mkRun(t, w, id, detachedRunningPlaceholder+"\nchild: a-codex\ntask: t-1\n")
		if err := os.WriteFile(filepath.Join(dir, "transcript.log"), []byte("{\"type\":\"item.started\"}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{
			RunID: id, Child: fmt.Sprintf("a-codex-%d", i+1), Task: "t-1",
			PID: 1 << 30, PGID: 1 << 30, Started: time.Now().Add(-age),
		}); err != nil {
			t.Fatal(err)
		}
	}

	ctx, agents, _ := newCtx(w.Root)
	if err := cmdAgents(ctx, nil); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		if !strings.Contains(agents.String(), runID(i)[:10]) {
			t.Fatalf("agents omitted actively starting run %s:\n%s", runID(i), agents)
		}
	}
	if !strings.Contains(agents.String(), "STARTUP-GRACE") {
		t.Fatalf("agents does not make bounded startup liveness observable:\n%s", agents)
	}

	ctx, runs, _ := newCtx(w.Root)
	if err := cmdRunsList(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(runs.String(), detachedRunningPlaceholder); got != 2 {
		t.Fatalf("runs list disagrees with startup lifecycle: got %d running entries\n%s", got, runs)
	}

	ctx, _, _ = newCtx(w.Root)
	err := cmdWait(ctx, []string{runID(1), runID(2), "--interval", "1", "--timeout", "1"})
	if err == nil || !strings.Contains(err.Error(), "2 run(s) still live") {
		t.Fatalf("wait finalized active startup runs, want bounded timeout retaining both: %v", err)
	}
	for i := 1; i <= 2; i++ {
		raw, readErr := os.ReadFile(filepath.Join(w.RunDir(runID(i)), "outcome.md"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.Contains(string(raw), detachedRunningPlaceholder) {
			t.Fatalf("wait finalized actively starting run %s:\n%s", runID(i), raw)
		}
	}
}

func TestRunLifecycleLivenessBoundsTranscriptActivity(t *testing.T) {
	w := newExecWS(t)
	id := runID(1)
	dir := mkRun(t, w, id, detachedRunningPlaceholder+"\n")
	transcript := filepath.Join(dir, "transcript.log")
	if err := os.WriteFile(transcript, []byte("advancing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{RunID: id, PID: 1 << 30, Started: time.Now().Add(-runStartupGrace - time.Second)}
	if live, reason := runLifecycleLive(w, rec, time.Now()); !live || reason != "transcript active" {
		t.Fatalf("fresh transcript liveness = (%v, %q), want (true, transcript active)", live, reason)
	}
	stale := time.Now().Add(-transcriptActiveGrace - time.Second)
	if err := os.Chtimes(transcript, stale, stale); err != nil {
		t.Fatal(err)
	}
	if live, reason := runLifecycleLive(w, rec, time.Now()); live || reason != "" {
		t.Fatalf("stale dead launch liveness = (%v, %q), want (false, empty)", live, reason)
	}
}

func TestRunLifecycleDurableKillEndsTranscriptLeaseImmediately(t *testing.T) {
	w := newExecWS(t)
	id := runID(1)
	dir := mkRun(t, w, id, detachedRunningPlaceholder+"\n")
	if err := os.WriteFile(filepath.Join(dir, "transcript.log"), []byte("work before the governed kill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "killed.txt"), []byte("process tree reaped\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{
		RunID: id, Child: "a-killed", Task: "t-killed", PID: 1 << 30, PGID: 1 << 30,
		Started: time.Now().Add(-time.Minute), Timeout: 2 * time.Hour,
		Claims: []string{"internal/features/execution"},
	}
	procPath := filepath.Join(dir, "proc.txt")
	if err := procmon.WriteRecord(procPath, rec); err != nil {
		t.Fatal(err)
	}

	if live, reason := runLifecycleLive(w, rec, time.Now()); live || reason != "" {
		t.Fatalf("durably killed run retained transcript lease: live=%v reason=%q", live, reason)
	}
	ctx, _, _ := newCtx(w.Root)
	if err := cmdWait(ctx, []string{id, "--interval", "1", "--timeout", "1"}); err != nil {
		t.Fatalf("wait did not finalize durably killed run: %v", err)
	}
	finished, err := procmon.ReadRecord(procPath)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Outcome == "" || len(finished.Claims) != 0 {
		t.Fatalf("durably killed run did not release claims: %+v", finished)
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

// Task 382: a status reader may be unable to authenticate a process that is
// genuinely alive (for example, ps is restricted by an outer sandbox). That
// false-negative probe is not evidence that the run exited, so neither agents
// --tail nor runs list may turn the detached placeholder into a final outcome.
// wait remains the lifecycle owner and may later finalize the child's durable
// result.
func TestStatusReadsDoNotFinalizeALiveRunWhenProcessIdentityIsHidden(t *testing.T) {
	w := newExecWS(t)
	id := runID(1)
	dir := mkRun(t, w, id, detachedRunningPlaceholder+"\nchild: a-hidden\ntask: t-1\n")
	rec := procmon.Record{
		RunID: id, Child: "a-hidden", Task: "t-1", Runtime: "rt",
		PID: os.Getpid(), PGID: os.Getpid(),
		// A caller that can signal the guardian but cannot observe its start
		// identity gets this same false-negative from ReconcileRun.
		PIDStart: "identity-not-visible-to-this-caller", Started: time.Now(),
	}
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	if !procmon.Alive(rec.PID) {
		t.Fatal("test guardian must remain alive")
	}
	if runStillLive(rec) {
		t.Fatal("test premise requires a false-negative identity probe")
	}

	want, err := os.ReadFile(filepath.Join(dir, "outcome.md"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	if err := cmdAgents(ctx, []string{"--tail"}); err != nil {
		t.Fatal(err)
	}
	ctx, _, _ = newCtx(w.Root)
	if err := cmdRunsList(ctx, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "outcome.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("status reads finalized a run whose guardian is alive:\n%s", got)
	}
	if evs := exitEvents(t, w); len(evs) != 0 {
		t.Fatalf("status reads recorded %d run exit event(s)", len(evs))
	}

	// The child subsequently reports its real durable result. wait is allowed
	// to reconcile lifecycle state and must preserve that outcome.
	if err := os.WriteFile(filepath.Join(dir, "blocked.txt"), []byte("runtime lost process visibility\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, _, _ = newCtx(w.Root)
	if err := cmdWait(ctx, []string{id}); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(filepath.Join(dir, "outcome.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "outcome: blocked (detached)") {
		t.Fatalf("wait did not finalize the child's real outcome:\n%s", got)
	}
}

func exitEvents(t *testing.T, w *workspace.Workspace) []*eventlog.Event {
	t.Helper()
	evs, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventExit}})
	if err != nil {
		t.Fatal(err)
	}
	return evs
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
	if err := cmdRuntimeAdd(ctx, []string{"cc", "--preset", "claude-code", "--model-flag", "--model-name", "--token-limit-flag", "--max-output"}); err != nil {
		t.Fatal(err)
	}
	rt, err := store.LoadRuntime(w, "cc")
	if err != nil {
		t.Fatal(err)
	}
	if rt.Binary != "claude" || rt.Harness != "claude" || rt.Mode != "arg" || rt.Flag != "-p" {
		t.Errorf("preset fields lost: %+v", rt)
	}
	if rt.ModelFlag != "--model-name" {
		t.Errorf("--model-flag override lost: %q", rt.ModelFlag)
	}
	if rt.TokenLimitFlag != "--max-output" {
		t.Errorf("--token-limit-flag override lost: %q", rt.TokenLimitFlag)
	}
	if rt.UsageFormat != "stream-json" {
		t.Errorf("the claude-code preset must default usage_format to stream-json, got %q", rt.UsageFormat)
	}
	// The shipped preset must never forward a credential.
	if bad := deniedEnvPassthrough(rt.Env); bad != "" {
		t.Errorf("the claude-code preset forwards a denied env %q", bad)
	}
	for _, preset := range []string{"gemini", "gemini-rw", "copilot", "copilot-rw"} {
		ctx, _, _ := newCtx(w.Root)
		if err := cmdRuntimeAdd(ctx, []string{preset, "--preset", preset}); err != nil {
			t.Fatalf("runtime add --preset %s: %v", preset, err)
		}
		got, err := store.LoadRuntime(w, preset)
		if err != nil {
			t.Fatal(err)
		}
		if got.Harness != strings.TrimSuffix(preset, "-rw") || got.ModelFlag != "--model" || len(got.SandboxRO) == 0 {
			t.Errorf("runtime add lost %s contract: %+v", preset, got)
		}
	}

	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRuntimeAdd(ctx2, []string{"x", "--preset", "no-such-preset"})); code != 2 {
		t.Error("an unknown preset must be a usage error")
	}
}

func TestCodexPresetsMatchExecCLIOrdering(t *testing.T) {
	for _, name := range []string{"codex", "codex-rw"} {
		rt, ok := presets[name]
		if !ok {
			t.Fatalf("missing %s preset", name)
		}
		if got := strings.Join(rt.GlobalArgs, " "); got != "--ask-for-approval never" {
			t.Errorf("%s globals = %q", name, got)
		}
		if len(rt.Args) == 0 || rt.Args[0] != "exec" {
			t.Errorf("%s exec ordering: %v", name, rt.Args)
		}
		joined := strings.Join(rt.Args, " ")
		for _, want := range []string{"--json", "--ephemeral"} {
			if !strings.Contains(joined, want) {
				t.Errorf("%s missing %s: %v", name, want, rt.Args)
			}
		}
		if rt.Harness != "codex" || rt.Mode != "stdin" || rt.ModelFlag != "--model" || rt.UsageFormat != "codex-jsonl" || rt.BehavioralPreflight != store.BehavioralPreflightCodexExecJSONV2 {
			t.Errorf("bad %s preset: %+v", name, rt)
		}
	}
}

func TestRuntimeDoctorCodexProbeRequiresWriteRefusal(t *testing.T) {
	w := newExecWS(t)
	dir := t.TempDir()
	fake := filepath.Join(dir, "codex")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'codex-cli 0.147.0'; exit 0; fi\nif [ \"$1 $2 $3\" = 'sandbox -P :read-only' ]; then echo 'touch: must-not-write: Operation not permitted' >&2; echo 'dacli-codex-ro-command-ran:1'; exit 1; fi\nexit 2\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRuntime(w, "test", store.Runtime{Name: "codex", Binary: fake, SandboxRO: []string{"--sandbox", "read-only"}}, ""); err != nil {
		t.Fatal(err)
	}
	ctx, out, _ := newCtx(w.Root)
	if err := cmdRuntimeDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "local codex sandbox refused a write") {
		t.Fatalf("doctor output:\n%s", out.String())
	}
	rt, _ := store.LoadRuntime(w, "codex")
	if got := store.HydrateRuntimeROProbe(w, rt, fake).ROProbe; got != store.RuntimeROVerified {
		t.Errorf("probe = %s", got)
	}
}

// A setup/usage error also exits non-zero without creating the sentinel. That
// is not evidence of isolation: doctor must require an OS sandbox denial from
// the attempted command before it caches a verified verdict.
func TestRuntimeDoctorCodexProbeRejectsSetupFailure(t *testing.T) {
	w := newExecWS(t)
	dir := t.TempDir()
	fake := filepath.Join(dir, "codex")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'codex-cli 0.147.0'; exit 0; fi\necho 'error: unknown permission profile' >&2\nexit 2\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRuntime(w, "test", store.Runtime{Name: "codex", Binary: fake, SandboxRO: []string{"--sandbox", "read-only"}}, ""); err != nil {
		t.Fatal(err)
	}
	ctx, out, _ := newCtx(w.Root)
	if err := cmdRuntimeDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "sandbox probe failed") {
		t.Fatalf("doctor accepted a setup failure as isolation:\n%s", out.String())
	}
	rt, _ := store.LoadRuntime(w, "codex")
	if got := store.HydrateRuntimeROProbe(w, rt, fake).ROProbe; got != store.RuntimeROFailed {
		t.Errorf("probe = %s, want failed", got)
	}
}

func TestRuntimeDoctorCodexProbeRejectsOuterSandboxStartupFailure(t *testing.T) {
	w := newExecWS(t)
	dir := t.TempDir()
	fake := filepath.Join(dir, "codex")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'codex-cli 0.147.0'; exit 0; fi\necho 'sandbox-exec: sandbox_apply: Operation not permitted' >&2\nexit 71\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRuntime(w, "test", store.Runtime{Name: "codex", Binary: fake, SandboxRO: []string{"--sandbox", "read-only"}}, ""); err != nil {
		t.Fatal(err)
	}
	ctx, out, _ := newCtx(w.Root)
	if err := cmdRuntimeDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "sandbox verified") {
		t.Fatalf("doctor false-verified an outer sandbox startup failure:\n%s", out.String())
	}
	rt, _ := store.LoadRuntime(w, "codex")
	if got := store.HydrateRuntimeROProbe(w, rt, fake).ROProbe; got == store.RuntimeROVerified {
		t.Fatalf("outer startup failure persisted probe = %s", got)
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

func TestGeminiAndCopilotPresetsAreLeastPrivilege(t *testing.T) {
	tests := []struct {
		name, binary, format string
		ro, rw               []string
	}{
		{"gemini", "gemini", "gemini-stream-json", []string{"--approval-mode", "plan"}, []string{"--approval-mode", "auto_edit"}},
		{"copilot", "copilot", "copilot-json", []string{"--deny-tool", "write", "--deny-tool", "shell"}, []string{"--allow-tool", "write", "--allow-tool", "shell(git:*)", "--allow-tool", "shell(dacli:*)"}},
	}
	for _, tc := range tests {
		for _, suffix := range []string{"", "-rw"} {
			rt, ok := presets[tc.name+suffix]
			if !ok {
				t.Errorf("missing preset %s%s", tc.name, suffix)
				continue
			}
			if rt.Binary != tc.binary || rt.Mode != "arg" || rt.Flag != "-p" || rt.ModelFlag != "--model" || rt.UsageFormat != tc.format {
				t.Errorf("bad preset %s%s: %+v", tc.name, suffix, rt)
			}
			if strings.Join(rt.SandboxRO, "\x00") != strings.Join(tc.ro, "\x00") {
				t.Errorf("%s%s ro args = %v, want %v", tc.name, suffix, rt.SandboxRO, tc.ro)
			}
			if suffix == "-rw" && strings.Join(rt.Args, "\x00") != strings.Join(tc.rw, "\x00") {
				t.Errorf("%s-rw args = %v, want %v", tc.name, rt.Args, tc.rw)
			}
		}
	}
}

func TestRuntimeDoctorVendorPresetFlagDriftFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name, help string
	}{
		{"gemini", "--prompt --model --output-format --approval-mode plan"},
		{"copilot", "--prompt --model --output-format --deny-tool"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newExecWS(t)
			dir := t.TempDir()
			fake := filepath.Join(dir, tc.name)
			script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo '" + tc.name + " 1.0.0'; exit 0; fi\nprintf '%s\\n' '" + tc.help + "'\n"
			if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			rt := presets[tc.name]
			rt.Name, rt.Binary = tc.name, fake
			if err := store.CreateRuntime(w, "test", rt, ""); err != nil {
				t.Fatal(err)
			}
			ctx, out, _ := newCtx(w.Root)
			if err := cmdRuntimeDoctor(ctx, nil); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "sandbox verified") {
				t.Fatalf("valid help was not verified:\n%s", out.String())
			}

			script = strings.ReplaceAll(script, " --model", "")
			if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			ctx, out, _ = newCtx(w.Root)
			if err := cmdRuntimeDoctor(ctx, nil); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "sandbox probe failed") {
				t.Fatalf("changed flags were trusted:\n%s", out.String())
			}
			loaded, _ := store.LoadRuntime(w, tc.name)
			if got := store.HydrateRuntimeROProbe(w, loaded, fake).ROProbe; got != store.RuntimeROFailed {
				t.Fatalf("drift probe = %s, want failed", got)
			}
		})
	}
}
