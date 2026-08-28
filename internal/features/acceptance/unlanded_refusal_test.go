package acceptance

import (
	"bytes"
	"fmt"
	"strings"
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

	err := acceptOne(ctx, w, root, task, "true", true, false, false, false, false, "")
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

	err := acceptOne(ctx, w, root, task, "true", true, false, false, true, false, "")
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
	// The proposal is durable workspace input to the owner, not source under
	// test. Commit it so acceptance-grade verification sees the clean immutable
	// tree this fixture intends to exercise.
	git(t, w.Root, "-C", w.Root, "add", "-A")
	git(t, w.Root, "-C", w.Root, "commit", "-q", "-m", "record proposal")

	err := acceptAll(ctx, w, root, "true", false, true, false, false, false, false, "")
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
	git(t, w.Root, "-C", w.Root, "add", "-A")
	git(t, w.Root, "-C", w.Root, "commit", "-q", "-m", "record proposal")

	if err := acceptAll(ctx, w, root, "true", false, true, false, false, true, false, ""); err != nil {
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

// --- the DEFAULT path (issue #443) -------------------------------------
//
// Every test above passes requireVerify=true, so none of them ever exercised
// what happens without it — and without it this used to be a warning to
// stderr, followed by the close. The suite stayed green whichever way that
// branch went, which is how the behaviour survived: a warning is written for
// an operator who is watching, and the close it must stop happens inside a
// loop where nobody is.
//
// The reported failure: a task closed four seconds after its PR opened, boxes
// checked and a passing verify recorded, against work that was still not in
// main six days later and whose PR had become unmergeable.

// TestAcceptOneRefusesUnlandedByDefault is the #443 regression.
func TestAcceptOneRefusesUnlandedByDefault(t *testing.T) {
	w, task, ctx := unlandedTaskFixture(t)
	root := &agentid.Identity{ID: "a-root", Grant: model.GrantRW, Role: "root"}

	// requireVerify=false — the default, and no --verify command either.
	err := acceptOne(ctx, w, root, task, "", false, false, false, false, false, "")
	if err == nil {
		t.Fatal("acceptOne closed a task whose branch is NOT in trunk, with no flag asked for")
	}
	if got := clikit.ExitCode(err); got != 3 {
		t.Errorf("exit code = %d, want 3 — a supervisor that retries this cannot ever succeed", got)
	}
	if task.Status == model.StatusDone {
		t.Fatal("the task must stay open")
	}
	// The refusal has to say how to proceed deliberately, or the only way out
	// of it is to stop using the tool.
	if !strings.Contains(err.Error(), "--allow-unlanded") {
		t.Fatalf("the refusal must name the escape hatch, got: %v", err)
	}
}

// TestAcceptAllRefusesUnlandedByDefault covers the batch path, which is the one
// ship and the loop actually drive.
func TestAcceptAllRefusesUnlandedByDefault(t *testing.T) {
	w, task, ctx := unlandedTaskFixture(t)
	root := &agentid.Identity{ID: "a-root", Grant: model.GrantRW, Role: "root"}
	if err := propose(ctx, w, root, task); err != nil {
		t.Fatal(err)
	}

	err := acceptAll(ctx, w, root, "", false, false, false, false, false, false, "")
	if err == nil {
		t.Fatal("acceptAll closed a task whose branch is NOT in trunk, with no flag asked for")
	}
	if got := clikit.ExitCode(err); got != 3 {
		t.Errorf("exit code = %d, want 3", got)
	}
}

// TestAcceptOneStillClosesLandedWorkByDefault is what stops the change above
// from degenerating into "accept refuses everything": the ordinary case — work
// that IS in trunk — must close with no flags at all. Without this the two
// tests above would pass against a command nobody could use.
func TestAcceptOneStillClosesLandedWorkByDefault(t *testing.T) {
	w, task := landedFixture(t)
	branch := store.TaskBranch(task)
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "-b", branch)
	git(t, w.Root, "-C", w.Root, "commit", "-q", "--allow-empty", "-m", "the deliverable")
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "main")
	git(t, w.Root, "-C", w.Root, "merge", "-q", "--no-ff", "-m", "merge it", branch)
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	root := &agentid.Identity{ID: "a-root", Grant: model.GrantRW, Role: "root"}

	if err := acceptOne(ctx, w, root, task, "", false, false, false, false, false, ""); err != nil {
		t.Fatalf("work that IS in trunk must close with no flags: %v", err)
	}
	if task.Status != model.StatusDone {
		t.Fatalf("landed work did not close, status=%s", task.Status)
	}
}

// TestAcceptOneStillClosesABranchlessTaskByDefault: work committed straight to
// trunk, a docs task, a record task. There is no branch to contradict, so the
// refusal must not fire — otherwise the whole class becomes uncloseable.
func TestAcceptOneStillClosesABranchlessTaskByDefault(t *testing.T) {
	w, task := landedFixture(t)
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	root := &agentid.Identity{ID: "a-root", Grant: model.GrantRW, Role: "root"}

	if err := acceptOne(ctx, w, root, task, "", false, false, false, false, false, ""); err != nil {
		t.Fatalf("a task with no branch has nothing to contradict: %v", err)
	}
	if task.Status != model.StatusDone {
		t.Fatalf("branchless task did not close, status=%s", task.Status)
	}
}

// TestAcceptOneDeferLandingStillClosesUnlanded: ship passes --defer-landing
// because it is about to integrate the branch ITSELF and will record the real
// verdict afterwards. Checking now would only ever see "not yet landed", so
// refusing here would make ship unable to run at all.
func TestAcceptOneDeferLandingStillClosesUnlanded(t *testing.T) {
	w, task, ctx := unlandedTaskFixture(t)
	root := &agentid.Identity{ID: "a-root", Grant: model.GrantRW, Role: "root"}

	if err := acceptOne(ctx, w, root, task, "", false, false, false, false, true, ""); err != nil {
		t.Fatalf("--defer-landing must still close: ship integrates afterwards and records the truth then: %v", err)
	}
	if task.Status != model.StatusDone {
		t.Fatalf("--defer-landing did not close, status=%s", task.Status)
	}
}

// TestVerificationEvidenceStatesWhereItRan: `verified by <cmd> (exit 0)` reads
// as a claim about the deliverable and is not one — a build-and-test proves the
// tree it ran in compiles, which is a different sentence from "this work is in
// trunk". When it runs in the branch that just wrote the code, the two are not
// even close, and that is how a task was closed over work that never merged
// (issue #443). The record must not imply a broader verification than happened.
func TestVerificationEvidenceStatesWhereItRan(t *testing.T) {
	ev := verificationEvidence("go build ./... && go test ./...", verifyContext{Branch: "dacli/007-x", Head: "abc1234"})
	for _, want := range []string{"go build", "dacli/007-x", "abc1234"} {
		if !strings.Contains(ev, want) {
			t.Fatalf("the evidence must name the command AND the tree it ran in; %q missing from: %s", want, ev)
		}
	}
	// The claim must be bounded in the line itself, not left to the reader.
	if !strings.Contains(ev, "not that the work is in trunk") {
		t.Fatalf("the evidence must bound what it proves, got: %s", ev)
	}
}

// TestVerificationEvidenceIsHonestWhenTheTreeIsUnknown: a non-git or unreadable
// tree must produce an explicitly unidentified context, never a confident
// blank that reads like a verified trunk.
func TestVerificationEvidenceIsHonestWhenTheTreeIsUnknown(t *testing.T) {
	ev := verificationEvidence("make check", verifyContext{})
	if !strings.Contains(ev, "unidentified") {
		t.Fatalf("an unknown tree must say so, got: %s", ev)
	}
	if !strings.Contains(ev, "nothing about trunk") {
		t.Fatalf("an unknown tree must claim nothing, got: %s", ev)
	}
}

// TestVerificationEvidenceStillDistinguishesNoVerification is the older
// guarantee (dacli 184) — an unverified close must never look like a verified
// one — kept explicit so the rewrite above cannot quietly drop it.
func TestVerificationEvidenceStillDistinguishesNoVerification(t *testing.T) {
	ev := verificationEvidence("", verifyContext{Branch: "main", Head: "deadbee"})
	if !strings.Contains(ev, "WITHOUT verification") {
		t.Fatalf("an unverified close must say so plainly, got: %s", ev)
	}
	if strings.Contains(ev, "main") || strings.Contains(ev, "deadbee") {
		t.Fatalf("with no command there is no tree to credit, got: %s", ev)
	}
}
