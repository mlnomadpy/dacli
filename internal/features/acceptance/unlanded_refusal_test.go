package acceptance

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// unlandedTaskFixture is landedFixture (a real repo, so the landing check runs
// against actual git history) plus a branch carrying a commit that never
// reached trunk — the exact shape issue #382 reported (done:15/21 while the
// commands did not exist because the PR never merged).
func unlandedTaskFixture(t *testing.T) (*workspace.Workspace, *store.Task, *clikit.Ctx) {
	t.Helper()
	w, task := landedFixture(t)
	branch := store.TaskBranch(task)
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "-b", branch)
	git(t, w.Root, "-C", w.Root, "commit", "-q", "--allow-empty", "-m", "the deliverable")
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "main")
	return w, task, &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
}

// TestAcceptOneRefusesUnlandedUnderRequireVerify is the acceptOne half of the
// close refusal: a task whose branch carries commits NOT in trunk must be
// refused (exit 3) under --require-verify, not closed on an assertion the
// trunk cannot back up.
func TestAcceptOneRefusesUnlandedUnderRequireVerify(t *testing.T) {
	w, task, ctx := unlandedTaskFixture(t)
	root := &agentid.Identity{ID: "a-root", Grant: model.GrantRW, Role: "root"}

	err := acceptOne(ctx, w, root, task, "true", true, false, false, false, false)
	if err == nil {
		t.Fatal("acceptOne closed a task whose branch is NOT in trunk under --require-verify")
	}
	if got := clikit.ExitCode(err); got != 3 {
		t.Errorf("exit code = %d, want 3 (policy refusal)", got)
	}
	if task.Status == model.StatusDone {
		t.Fatal("task must stay open when the unlanded refusal fires")
	}
}

// TestAcceptOneAllowUnlandedStillCloses proves the refusal above is the
// --allow-unlanded flag's doing, not a blanket refusal on every unlanded
// branch: with the flag set, the same fixture closes.
func TestAcceptOneAllowUnlandedStillCloses(t *testing.T) {
	w, task, ctx := unlandedTaskFixture(t)
	root := &agentid.Identity{ID: "a-root", Grant: model.GrantRW, Role: "root"}

	err := acceptOne(ctx, w, root, task, "true", true, false, false, true, false)
	if err != nil {
		t.Fatalf("--allow-unlanded must still close the task: %v", err)
	}
	if task.Status != model.StatusDone {
		t.Fatalf("--allow-unlanded did not close the task, status=%s", task.Status)
	}
}

// TestAcceptAllRefusesUnlandedUnderRequireVerify is the acceptAll half — the
// batch path `ship` and the loop drive. A proposed task whose branch never
// reached trunk must refuse (exit 3) under --require-verify, exactly like the
// single-ref path.
func TestAcceptAllRefusesUnlandedUnderRequireVerify(t *testing.T) {
	w, task, ctx := unlandedTaskFixture(t)
	root := &agentid.Identity{ID: "a-root", Grant: model.GrantRW, Role: "root"}
	if err := propose(ctx, w, root, task); err != nil {
		t.Fatal(err)
	}

	err := acceptAll(ctx, w, root, "true", false, true, false, false, false, false)
	if err == nil {
		t.Fatal("acceptAll closed a task whose branch is NOT in trunk under --require-verify")
	}
	if got := clikit.ExitCode(err); got != 3 {
		t.Errorf("exit code = %d, want 3 (policy refusal)", got)
	}
	ref := fmt.Sprintf("%03d", task.Seq)
	got, err := store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == model.StatusDone {
		t.Fatal("task must stay open when the unlanded refusal fires")
	}
}

// TestAcceptAllAllowUnlandedStillCloses is acceptAll's sibling to
// TestAcceptOneAllowUnlandedStillCloses: --allow-unlanded closes the batch
// path's task too, proving the refusal is flag-gated there as well.
func TestAcceptAllAllowUnlandedStillCloses(t *testing.T) {
	w, task, ctx := unlandedTaskFixture(t)
	root := &agentid.Identity{ID: "a-root", Grant: model.GrantRW, Role: "root"}
	if err := propose(ctx, w, root, task); err != nil {
		t.Fatal(err)
	}

	if err := acceptAll(ctx, w, root, "true", false, true, false, false, true, false); err != nil {
		t.Fatalf("--allow-unlanded must still close the task: %v", err)
	}
	ref := fmt.Sprintf("%03d", task.Seq)
	got, err := store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusDone {
		t.Fatalf("--allow-unlanded did not close the task, status=%s", got.Status)
	}
}
