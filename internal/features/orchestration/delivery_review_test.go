package orchestration

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

type reviewTransactionRunner struct {
	fakeRunner
	w        *workspace.Workspace
	task     *store.Task
	reviews  int
	noOutput bool
}

func (r *reviewTransactionRunner) runResult(label string, target any, args ...string) (string, error) {
	_, _ = r.fakeRunner.run(label, args...)
	spawn, ok := target.(*commandresult.Spawn)
	if !ok {
		return "", fmt.Errorf("unexpected result target %T", target)
	}
	spawn.RunID = fmt.Sprintf("01REVIEWRUN%014d", r.reviews+1)
	if label != "delivery-review" || r.noOutput {
		return "", nil
	}
	r.reviews++
	commit, _ := gitx.Run(r.w.Root, "rev-parse", taskBranch(r.task))
	commit = strings.TrimSpace(commit)
	tree, _ := gitx.Run(r.w.Root, "rev-parse", commit+"^{tree}")
	verdict := store.ReviewRequestChanges
	var findings []store.ReviewFinding
	if r.reviews == 1 {
		findings = []store.ReviewFinding{{ID: "REV-001", Severity: "major", File: "source.go", Line: 1, Evidence: "observable defect", AffectedInvariant: "source must be corrected", SuggestedVerification: "go test ./..."}}
	} else {
		verdict = store.ReviewApprove
	}
	result := store.IndependentReviewResult{Schema: store.ReviewResultSchema, Verdict: verdict, Findings: findings, ReviewerID: fmt.Sprintf("a-reviewer-%d", r.reviews), ReviewerRole: "reviewer", Runtime: "codex", Grant: "ro", IndependentOf: []string{"a-builder"}, CommitSHA: commit, TreeSHA: strings.TrimSpace(tree), ObservedAt: time.Now()}
	raw, _ := json.Marshal(result)
	_, err := eventlog.Append(r.w, result.ReviewerID, model.EventReview, r.task.ID, reviewEventOrigin, string(raw))
	return "", err
}

func (r *reviewTransactionRunner) run(label string, args ...string) (string, error) {
	_, _ = r.fakeRunner.run(label, args...)
	if label != "review-correction" {
		return "", nil
	}
	parent, err := gitx.Run(r.w.Root, "rev-parse", taskBranch(r.task))
	if err != nil {
		return "", err
	}
	blob := exec.Command("git", "hash-object", "-w", "--stdin")
	blob.Dir, blob.Stdin = r.w.Root, strings.NewReader("corrected\n")
	blobOut, err := blob.Output()
	if err != nil {
		return "", err
	}
	mktree := exec.Command("git", "mktree")
	mktree.Dir, mktree.Stdin = r.w.Root, strings.NewReader("100644 blob "+strings.TrimSpace(string(blobOut))+"\tsource.go\n")
	treeOut, err := mktree.Output()
	if err != nil {
		return "", err
	}
	commit := exec.Command("git", "commit-tree", strings.TrimSpace(string(treeOut)), "-p", strings.TrimSpace(parent), "-m", "review correction")
	commit.Dir = r.w.Root
	commitOut, err := commit.Output()
	if err != nil {
		return "", err
	}
	_, err = gitx.Run(r.w.Root, "update-ref", "refs/heads/"+taskBranch(r.task), strings.TrimSpace(string(commitOut)))
	return "", err
}

func reviewedTaskFixture(t *testing.T) (*workspace.Workspace, *store.Task) {
	t.Helper()
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Review transaction", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-q", "-m", "base"}, {"branch", taskBranch(task)}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = w.Root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return w, task
}

func TestDeliveryReviewRunsBoundedCorrectionAndFreshRereview(t *testing.T) {
	w, task := reviewedTaskFixture(t)
	runner := &reviewTransactionRunner{w: w, task: task}
	d := newDriver(w, runner, &Governor{})
	d.phases = cyclePhaseJournal{Project: "p"}
	if !d.reviewDeliveryTask(task) {
		t.Fatal("fresh re-review did not approve corrected tree")
	}
	if runner.reviews != 2 {
		t.Fatalf("reviews=%d, want initial plus re-review", runner.reviews)
	}
	tx, err := store.ReadReviewTransaction(w, "p", task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tx.State != store.ReviewApproved || tx.CorrectionTurns != 1 || tx.PriorTree == tx.CurrentTree {
		t.Fatalf("transaction lost correction/re-review binding: %+v", tx)
	}
	var correction, rereview bool
	for _, call := range runner.calls {
		correction = correction || len(call) > 0 && call[0] == "spawn" && !contains(call, "--review")
		rereview = rereview || len(call) > 0 && call[0] == "spawn" && contains(call, "--review")
	}
	if !correction || !rereview {
		t.Fatalf("missing correction/re-review calls: %v", runner.calls)
	}
}

func TestDeliveryReviewMissingOutputNeverApproves(t *testing.T) {
	w, task := reviewedTaskFixture(t)
	runner := &reviewTransactionRunner{w: w, task: task, noOutput: true}
	d := newDriver(w, runner, &Governor{})
	d.phases = cyclePhaseJournal{Project: "p"}
	if d.reviewDeliveryTask(task) {
		t.Fatal("missing structured reviewer output counted as approval")
	}
	tx, err := store.ReadReviewTransaction(w, "p", task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tx.State != store.ReviewHalted {
		t.Fatalf("missing output state=%s, want halted", tx.State)
	}
}

func TestDeliveryReviewUsesConfiguredCorrectionLimit(t *testing.T) {
	w, _ := reviewedTaskFixture(t)
	p, err := defaultProfile("p", "task")
	if err != nil {
		t.Fatal(err)
	}
	p.Verification.Commands = []string{"go test ./..."}
	p.Verification.CorrectionTurns = 1
	if err := saveProfile(w, p); err != nil {
		t.Fatal(err)
	}
	d := newDriver(w, &fakeRunner{}, &Governor{})
	if got := d.reviewCorrectionLimit(); got != 1 {
		t.Fatalf("correction limit=%d, want configured 1", got)
	}
}
