package planning

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// unsetAgentEnv clears DACLI_AGENT for the test, restoring whatever the
// process started with. t.Setenv cannot unset a variable, and since dacli 288
// a present-but-empty DACLI_AGENT is a lost token that fails closed rather
// than resolving to root — so a test wanting the root identity must remove
// the variable entirely, not blank it.
func unsetAgentEnv(t *testing.T) {
	t.Helper()
	if v, ok := os.LookupEnv("DACLI_AGENT"); ok {
		t.Setenv("DACLI_AGENT", v)
		_ = os.Unsetenv("DACLI_AGENT")
	}
}

// taskAddEnv builds a workspace with one project and returns a Ctx ready to
// drive cmdTaskAdd directly. DACLI_AGENT is cleared so the acting identity is
// root regardless of who runs the suite.
func taskAddEnv(t *testing.T) (*workspace.Workspace, *clikit.Ctx) {
	t.Helper()
	unsetAgentEnv(t)
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	return w, &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
}

func TestTaskAddRejectsNonFiniteEstimateWithoutCreatingTask(t *testing.T) {
	w, ctx := taskAddEnv(t)
	err := cmdTaskAdd(ctx, []string{"impossible estimate", "--project", "p", "--estimate", "Inf,Inf,Inf"})
	if clikit.ExitCode(err) != 2 {
		t.Fatalf("exit = %d, want 2 (usage): %v", clikit.ExitCode(err), err)
	}
	tasks, listErr := store.ListTasks(w, "p", "")
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(tasks) != 0 {
		t.Fatalf("refused task add persisted %d task(s)", len(tasks))
	}
}

func TestTaskTakeoverRefusesLiveOwnerOrTranscriptActiveRun(t *testing.T) {
	for _, tc := range []struct {
		name string
		live bool
	}{
		{name: "owner process live", live: true},
		{name: "task transcript active"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, ctx := taskAddEnv(t)
			task, err := store.CreateTask(w, "a-deadchild", "p", "Recoverable only after lease ends", store.TaskOpts{Accept: []string{"x"}})
			if err != nil {
				t.Fatal(err)
			}
			runID := "01TESTTAKEOVERLEASE"
			runDir := w.RunDir(runID)
			if err := os.MkdirAll(runDir, 0o755); err != nil {
				t.Fatal(err)
			}
			rec := procmon.Record{RunID: runID, Child: task.Owner(), Task: task.ID, Started: time.Now()}
			if tc.live {
				rec.PID = os.Getpid()
				rec.PGID = os.Getpid()
				rec.PIDStart, _ = procmon.ProcStart(rec.PID)
			} else if err := os.WriteFile(filepath.Join(runDir, "transcript.log"), []byte("still working\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), rec); err != nil {
				t.Fatal(err)
			}

			err = cmdTaskTakeover(ctx, []string{task.ID, "--force", "--reason", "operator reviewed recovery"})
			if clikit.ExitCode(err) != 3 {
				t.Fatalf("takeover exit = %d, want refusal 3: %v", clikit.ExitCode(err), err)
			}
			got, err := store.FindTask(w, task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Owner() != "a-deadchild" {
				t.Fatalf("refused takeover changed owner to %q", got.Owner())
			}
		})
	}
}

// TestTaskAddRefusesNearDuplicateOfOpenTask reproduces the dacli task 116
// incident: a review auditor re-filing an already-queued issue under
// slightly different wording must be refused (exit 3), not silently allowed
// to create backlog churn.
func TestTaskAddRefusesNearDuplicateOfOpenTask(t *testing.T) {
	w, ctx := taskAddEnv(t)
	if err := cmdTaskAdd(ctx, []string{"charge idle-cycle review spawns to the token window", "--project", "p"}); err != nil {
		t.Fatalf("first add: %v", err)
	}

	err := cmdTaskAdd(ctx, []string{"charge idle-cycle reviewer tokens to the --window-tokens budget", "--project", "p"})
	if err == nil {
		t.Fatal("near-duplicate title was accepted, want refusal")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit code = %d, want 3 (refusal)", clikit.ExitCode(err))
	}

	ts, lerr := store.ListTasks(w, "p", "")
	if lerr != nil {
		t.Fatal(lerr)
	}
	if len(ts) != 1 {
		t.Errorf("project has %d tasks after refused dup, want 1", len(ts))
	}
}

func TestTaskAddRefusesContentDuplicateWithoutAllocatingSequence(t *testing.T) {
	w, ctx := taskAddEnv(t)
	if err := cmdTaskAdd(ctx, []string{
		"Use task ULIDs for generated mutating commands", "--project", "p",
		"--context", "alpha task 001 and beta task 001 can target the wrong task in internal/features/execution/execution.go",
		"--accept", "generated mutating commands contain the stable task ULID",
		"--accept", "the alpha/001 beta/001 cross-project mutation regression passes",
	}); err != nil {
		t.Fatal(err)
	}
	existing, err := store.FindTask(w, "001")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, existing, model.StatusActive); err != nil {
		t.Fatal(err)
	}

	err = cmdTaskAdd(ctx, []string{
		"Fix generated worker prompts using ambiguous numeric refs for mutations", "--project", "p",
		"--context", "internal/features/execution/execution.go sends alpha/001 or beta/001 mutations to the wrong task",
		"--accept", "generated mutation commands use the stable task ULID",
		"--accept", "the alpha/001 beta/001 cross-project mutation regression passes",
	})
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("exit=%d err=%v, want refusal", clikit.ExitCode(err), err)
	}
	if !strings.Contains(err.Error(), existing.Slug) {
		t.Fatalf("refusal does not name existing task: %v", err)
	}

	tasks, err := store.ListTasks(w, "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("refusal mutated task store: %d tasks", len(tasks))
	}
	next, err := store.CreateTask(w, "a-root", "p", "A genuinely distinct task", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if next.Seq != 2 {
		t.Fatalf("refusal allocated sequence: next seq=%d, want 2", next.Seq)
	}
}

// TestTaskAddForceOverridesDedup confirms --force is the explicit, loud
// override — same shape as spawn/accept's --force — rather than a dead end.
func TestTaskAddForceOverridesDedup(t *testing.T) {
	w, ctx := taskAddEnv(t)
	if err := cmdTaskAdd(ctx, []string{"charge idle-cycle review spawns to the token window", "--project", "p"}); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := cmdTaskAdd(ctx, []string{"charge idle-cycle reviewer tokens to the --window-tokens budget", "--project", "p", "--force"}); err != nil {
		t.Fatalf("forced add: %v", err)
	}

	ts, err := store.ListTasks(w, "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 {
		t.Errorf("project has %d tasks after forced dup, want 2", len(ts))
	}
}

// TestTaskAddRejectsTypoedFlag reproduces the dacli 143 incident: a typo'd
// flag (--acccept instead of --accept) must fail loudly with exit 2 naming
// the offending flag, instead of ParseFlags silently dropping the caller's
// acceptance criterion and returning exit 0.
func TestTaskAddRejectsTypoedFlag(t *testing.T) {
	w, ctx := taskAddEnv(t)
	err := cmdTaskAdd(ctx, []string{"typo flag task", "--project", "p", "--acccept", "y"})
	if err == nil {
		t.Fatal("typo'd --acccept was accepted, want a usage error")
	}
	if clikit.ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", clikit.ExitCode(err))
	}

	ts, lerr := store.ListTasks(w, "p", "")
	if lerr != nil {
		t.Fatal(lerr)
	}
	if len(ts) != 0 {
		t.Errorf("project has %d tasks after rejected typo, want 0", len(ts))
	}

	if err := cmdTaskAdd(ctx, []string{"typo flag task", "--project", "p", "--accept", "y"}); err != nil {
		t.Fatalf("correctly spelled --accept: %v", err)
	}
}

// TestTaskDoneRefusesEmptyAcceptance is the dacli 289 regression: `task done`
// on a task whose Acceptance section is empty must be refused (exit 3), not
// closed. The unmet-box scan finds an empty list and would otherwise pass, so
// zero boxes read as all boxes and the task closed with nothing verified.
func TestTaskDoneRefusesEmptyAcceptance(t *testing.T) {
	w, ctx := taskAddEnv(t)
	tk, err := store.CreateTask(w, "a-root", "p", "nothing asked for", store.TaskOpts{})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	ref := fmt.Sprintf("%03d", tk.Seq)

	err = cmdTaskDone(ctx, []string{ref})
	if err == nil {
		t.Fatal("task done closed a task with an empty Acceptance section (dacli 289)")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit code = %d, want 3 (refused-by-policy)", clikit.ExitCode(err))
	}
	got, err := store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == model.StatusDone {
		t.Fatal("empty-acceptance task was moved to done despite the refusal")
	}
}

// TestTaskDoneAllowUnverifiedClosesEmptyAcceptance confirms the explicit escape
// hatch: --allow-unverified closes the criteria-less task but stamps the Log so
// the record says plainly that nothing was verified (dacli 289).
func TestTaskDoneAllowUnverifiedClosesEmptyAcceptance(t *testing.T) {
	w, ctx := taskAddEnv(t)
	tk, err := store.CreateTask(w, "a-root", "p", "chore with no criteria", store.TaskOpts{})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	ref := fmt.Sprintf("%03d", tk.Seq)

	if err := cmdTaskDone(ctx, []string{ref, "--allow-unverified"}); err != nil {
		t.Fatalf("--allow-unverified must close the task: %v", err)
	}
	got, err := store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusDone {
		t.Fatalf("--allow-unverified did not close the task, status=%s", got.Status)
	}
	logSec, _ := got.Doc.Section("Log")
	if !strings.Contains(logSec.Content, "UNVERIFIED") {
		t.Fatalf("an --allow-unverified close must record UNVERIFIED on the task Log; log=%q", logSec.Content)
	}
}

// TestTaskDoneClosesWhenAcceptanceMet is the positive sanity check: a task with
// its single criterion checked still closes normally — the 289 guard keys on
// the empty section only.
func TestTaskDoneClosesWhenAcceptanceMet(t *testing.T) {
	w, ctx := taskAddEnv(t)
	tk, err := store.CreateTask(w, "a-root", "p", "real work", store.TaskOpts{Accept: []string{"it works"}})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	store.CheckAllAcceptance(tk)
	if err := store.SaveTask(tk); err != nil {
		t.Fatal(err)
	}
	ref := fmt.Sprintf("%03d", tk.Seq)

	if err := cmdTaskDone(ctx, []string{ref}); err != nil {
		t.Fatalf("task done on a met task must close it: %v", err)
	}
	got, err := store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusDone {
		t.Fatalf("met task was not closed, status=%s", got.Status)
	}
}

func TestTaskCheckRefusesCommandCriterionWithoutProvenance(t *testing.T) {
	w, ctx := taskAddEnv(t)
	tk, err := store.CreateTask(w, "a-root", "p", "command checked work", store.TaskOpts{Accept: []string{"`go test ./...` exits zero"}})
	if err != nil {
		t.Fatal(err)
	}

	err = cmdTaskCheck(ctx, []string{fmt.Sprintf("%03d", tk.Seq), "--n", "1"})
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("task check exit = %d, want refusal for missing command evidence: %v", clikit.ExitCode(err), err)
	}
	got, findErr := store.FindTask(w, tk.Slug)
	if findErr != nil {
		t.Fatal(findErr)
	}
	if got.Acceptance()[0].Done {
		t.Fatal("command criterion was checked without provenance")
	}
}

func TestTaskCheckPersistsCommandVerificationProvenance(t *testing.T) {
	w, ctx := taskAddEnv(t)
	tk, err := store.CreateTask(w, "a-root", "p", "command checked work", store.TaskOpts{Accept: []string{"`printf artifact` exits zero"}})
	if err != nil {
		t.Fatal(err)
	}

	if err := cmdTaskCheck(ctx, []string{fmt.Sprintf("%03d", tk.Seq), "--n", "1", "--verify", "printf artifact"}); err != nil {
		t.Fatal(err)
	}
	got, err := store.FindTask(w, tk.Slug)
	if err != nil {
		t.Fatal(err)
	}
	records := store.VerificationEvidenceRecords(got)
	if len(records) != 1 || records[0].Command != "printf artifact" || records[0].Verifier != "a-root" || records[0].ArtifactHash == "" {
		t.Fatalf("incomplete task-check evidence: %#v", records)
	}
}

// TestProjectRmRefusesWithoutForce is the dacli task 118 recovery path: `rm`
// exists to delete a project created by mistake, but deleting a project takes
// its tasks/notes/risks/glossary with it, so it must refuse (exit 3) without
// --force rather than delete on the first try.
func TestProjectRmRefusesWithoutForce(t *testing.T) {
	w, ctx := taskAddEnv(t)
	if err := cmdTaskAdd(ctx, []string{"a task under p", "--project", "p", "--accept", "y"}); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	err := cmdProjectRm(ctx, []string{"p"})
	if err == nil {
		t.Fatal("rm without --force was accepted, want refusal")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit code = %d, want 3 (refusal)", clikit.ExitCode(err))
	}

	if _, lerr := store.LoadProject(w, "p"); lerr != nil {
		t.Errorf("project p was removed despite the refusal: %v", lerr)
	}
}

// TestProjectRmForceDeletesTheProject confirms --force actually removes the
// project directory and everything under it.
func TestProjectRmForceDeletesTheProject(t *testing.T) {
	w, ctx := taskAddEnv(t)
	if err := cmdTaskAdd(ctx, []string{"a task under p", "--project", "p", "--accept", "y"}); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	if err := cmdProjectRm(ctx, []string{"p", "--force"}); err != nil {
		t.Fatalf("forced rm: %v", err)
	}

	if _, lerr := store.LoadProject(w, "p"); lerr == nil {
		t.Error("project p still loads after forced rm")
	}
	ts, lerr := store.ListTasks(w, "", "")
	if lerr != nil {
		t.Fatal(lerr)
	}
	if len(ts) != 0 {
		t.Errorf("%d task(s) survived the project's deletion, want 0", len(ts))
	}
}

// TestProjectRmUnknownSlugNotFound confirms rm on a slug that never existed
// reports not-found (exit 4) rather than a silent no-op.
func TestProjectRmUnknownSlugNotFound(t *testing.T) {
	_, ctx := taskAddEnv(t)
	err := cmdProjectRm(ctx, []string{"nope", "--force"})
	if err == nil {
		t.Fatal("rm of an unknown slug was accepted, want not-found")
	}
	if clikit.ExitCode(err) != 4 {
		t.Errorf("exit code = %d, want 4 (not found)", clikit.ExitCode(err))
	}
}

// TestTaskAddAllowsUnrelatedTitles is the control: ordinary, distinct backlog
// titles must never be blocked by the dedup guard.
func TestTaskAddAllowsUnrelatedTitles(t *testing.T) {
	w, ctx := taskAddEnv(t)
	if err := cmdTaskAdd(ctx, []string{"fix flaky retry timer in the spawn watchdog", "--project", "p"}); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := cmdTaskAdd(ctx, []string{"document the SPM glossary term for slack", "--project", "p"}); err != nil {
		t.Fatalf("second add: %v", err)
	}

	ts, err := store.ListTasks(w, "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 {
		t.Errorf("project has %d tasks, want 2", len(ts))
	}
}
