package insight

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// doctorEnv builds a workspace holding one task owned by a *different* agent
// that never ran (no proc.txt was ever recorded for it) — the stand-in for a
// spawned child that has since finished and will never sync or accept.
func doctorEnv(t *testing.T) (*workspace.Workspace, *clikit.Ctx) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if v, ok := os.LookupEnv("DACLI_AGENT"); ok { // act as root
		t.Setenv("DACLI_AGENT", v)
		_ = os.Unsetenv("DACLI_AGENT")
	}
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "x@x"}, {"config", "user.name", "x"}, {"checkout", "-q", "-b", "main"}} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	w, err := workspace.Init(dir, "a-root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	return w, &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
}

// TestLessonMatchesTaskNeedsRealOverlap is the 248 regression test: a single
// shared word is noise (every lesson body is a paragraph, so one common word
// lands in almost all of them), and a substring hit is not a word at all. Only
// meaningful multi-word overlap should attach a lesson to a task.
func TestLessonMatchesTaskNeedsRealOverlap(t *testing.T) {
	task := &store.Task{Title: "Persist the governor window budget to disk", Slug: "persist-governor-window-budget"}

	cases := []struct {
		name  string
		l     store.Lesson
		match bool
	}{
		{
			// Shares only "budget" — one word. The old rule matched; it must not now.
			name:  "single shared word is not overlap",
			l:     store.Lesson{Title: "Burn alert per-run population mismatch", Body: "the ceiling counts only completing non-verify runs so the budget dilutes"},
			match: false,
		},
		{
			// Shares "governor" and "window" and "budget" — genuine topic overlap.
			name:  "multiple shared words attach",
			l:     store.Lesson{Title: "Governor window budget not persisted", Body: "the WindowTokens ceiling is not on disk; loopState overloads the governor window spent value"},
			match: true,
		},
		{
			// Task word "budget" is a substring of the lesson's "budgetary" but shares
			// no other real word — the old strings.Contains form matched, the set form
			// must not, and even a substring word-hit alone is below the two-word bar.
			name:  "substring-only hit does not attach",
			l:     store.Lesson{Title: "Fiscal budgetary controls", Body: "quarterly reporting on spending review across finance"},
			match: false,
		},
	}
	for _, tc := range cases {
		if got := lessonMatchesTask(tc.l, task); got != tc.match {
			t.Errorf("%s: lessonMatchesTask = %v, want %v", tc.name, got, tc.match)
		}
	}
}

func TestDoctorFlagsOrphanedTask(t *testing.T) {
	w, ctx := doctorEnv(t)
	tk, err := store.CreateTask(w, "a-deadchild", "p", "Orphaned work", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "orphaned-task") {
		t.Fatalf("expected orphaned-task finding, got:\n%s", out)
	}
	if !strings.Contains(out, "task takeover") {
		t.Fatalf("expected the audited takeover suggestion, got:\n%s", out)
	}
	if !strings.Contains(out, tk.Slug) {
		t.Fatalf("expected the orphaned task to be named, got:\n%s", out)
	}
}

func TestDoctorReportsUnreadableRecoveryEvidenceWithoutCallingTaskOrphaned(t *testing.T) {
	w, ctx := doctorEnv(t)
	tk, err := store.CreateTask(w, "a-deadchild", "p", "Owner state cannot be proven", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	runDir := w.RunDir("01TESTMALFORMEDRUN")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "proc.txt"), []byte("not a process record\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmdDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "recovery-evidence") || !strings.Contains(out, tk.Slug) {
		t.Fatalf("expected unreadable recovery evidence for %s, got:\n%s", tk.Slug, out)
	}
	if strings.Contains(out, "orphaned-task") {
		t.Fatalf("doctor must not call an indeterminate task orphaned, got:\n%s", out)
	}
}

// TestDoctorSkipsLoopAnchor is the 254 regression test: the standing
// continuous-improvement anchor is owned by "loop" (ensureImproveTask) and
// re-surveyed by a fresh auditor every cycle — "loop" is never a live process,
// so before the fix doctor flagged it orphaned on every run, training people to
// ignore doctor's output.
func TestDoctorSkipsLoopAnchor(t *testing.T) {
	w, ctx := doctorEnv(t)
	if _, err := store.CreateTask(w, "loop", "p",
		store.ContinuousImprovementMarker+": file the single highest-value evidence-based change",
		store.TaskOpts{Accept: []string{"filed a task"}}); err != nil {
		t.Fatal(err)
	}
	if err := cmdDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if strings.Contains(out, "orphaned-task") {
		t.Fatalf("the standing loop anchor must never be flagged as orphaned, got:\n%s", out)
	}
}

func TestDoctorSkipsRootOwnedTask(t *testing.T) {
	w, ctx := doctorEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Root's own work", store.TaskOpts{Accept: []string{"done"}}); err != nil {
		t.Fatal(err)
	}
	if err := cmdDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if strings.Contains(out, "orphaned-task") {
		t.Fatalf("root-owned task must never be flagged as orphaned, got:\n%s", out)
	}
}

// TestNextSkipsContinuousImprovementAnchor is the 112 regression test: the
// loop's readyTasks (orchestration.go) already excludes the standing
// "Continuous improvement" review anchor from what a builder gets handed;
// `dacli next` must agree, or a human reading it is told to "work on" a
// task that is never itself implementer work.
func TestNextSkipsContinuousImprovementAnchor(t *testing.T) {
	w, ctx := doctorEnv(t)
	anchor, err := store.CreateTask(w, "loop", "p", "Continuous improvement: file the single highest-value evidence-based change", store.TaskOpts{Accept: []string{"filed"}})
	if err != nil {
		t.Fatal(err)
	}
	real, err := store.CreateTask(w, "a-root", "p", "Real implementer work", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdNext(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if strings.Contains(out, anchor.Slug) {
		t.Fatalf("dacli next must never recommend the Continuous improvement anchor, got:\n%s", out)
	}
	if !strings.Contains(out, real.Slug) {
		t.Fatalf("dacli next should still recommend real implementer work, got:\n%s", out)
	}
}

// TestNextWithOnlyAnchorOpenReportsNoneReady covers the edge the loop already
// handles: when the anchor is the ONLY open task, next must not fall back to
// recommending it — there is simply nothing actionable.
func TestNextWithOnlyAnchorOpenReportsNoneReady(t *testing.T) {
	w, ctx := doctorEnv(t)
	anchor, err := store.CreateTask(w, "loop", "p", "Continuous improvement: file the single highest-value evidence-based change", store.TaskOpts{Accept: []string{"filed"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdNext(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if strings.Contains(out, anchor.Slug) {
		t.Fatalf("dacli next must never recommend the Continuous improvement anchor, got:\n%s", out)
	}
}

// TestLintExcludesDoneTasksFromOutput verifies that lint does not report
// findings about done tasks, even if they have ambiguous acceptance criteria.
func TestLintExcludesDoneTasksFromOutput(t *testing.T) {
	w, ctx := doctorEnv(t)

	// Create a done task with vague acceptance criteria (which would normally trigger lint)
	done, err := store.CreateTask(w, "a-root", "p", "Completed work", store.TaskOpts{Accept: []string{"should do something"}})
	if err != nil {
		t.Fatal(err)
	}
	// Move task to done status
	if err := store.MoveTask(w, done, model.StatusDone); err != nil {
		t.Fatal(err)
	}

	// Create an open task with vague acceptance criteria (which should trigger lint)
	open, err := store.CreateTask(w, "a-root", "p", "Actionable work", store.TaskOpts{Accept: []string{"should do something"}})
	if err != nil {
		t.Fatal(err)
	}

	// Run lint
	if err := cmdLint(ctx, []string{"--project", "p"}); err != nil {
		t.Fatal(err)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()

	// Done task's slug should never appear in lint output
	if strings.Contains(out, done.Slug) {
		t.Fatalf("done task %s must not appear in lint output, got:\n%s", done.Slug, out)
	}

	// Open task's slug should appear in lint output (it has vague acceptance)
	if !strings.Contains(out, open.Slug) {
		t.Fatalf("open task %s should appear in lint output, got:\n%s", open.Slug, out)
	}
}
