package acceptance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestAcceptanceProposalCommandsNeedNoManualTokenHandoff(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	task := mkTask(t, w, "root-owned command handoff")
	reviewerID, token, err := agentid.Spawn(w, root, "reviewer", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	runID := "run-command-reviewer"
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), procmon.Record{
		RunID: runID, Child: reviewerID, Task: task.ID, Role: "reviewer", Runtime: "codex", Started: time.Now(), Outcome: "success",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv(agentid.EnvVar, token)
	ctx.JSON = true
	if err := cmdAcceptPropose(ctx, []string{task.ID, "--verify", "true"}); err != nil {
		t.Fatal(err)
	}
	var proposed acceptanceProposalResult
	if err := json.Unmarshal(ctx.Stdout.(*bytes.Buffer).Bytes(), &proposed); err != nil {
		t.Fatalf("decode propose JSON: %v\n%s", err, ctx.Stdout)
	}
	if proposed.Proposal.ID == "" || proposed.Proposal.Grant != "ro" || proposed.Proposal.Runtime != "codex" || !strings.Contains(proposed.NextAction, proposed.Proposal.ID) || strings.Contains(proposed.NextAction, token) {
		t.Fatalf("proposal did not return a safe exact next action: %+v", proposed)
	}
	if err := os.Unsetenv(agentid.EnvVar); err != nil {
		t.Fatal(err)
	}
	ctx.Stdout.(*bytes.Buffer).Reset()
	ctx.Stderr.(*bytes.Buffer).Reset()
	if err := cmdAcceptApply(ctx, []string{task.ID, "--proposal", proposed.Proposal.ID, "--defer-landing"}); err != nil {
		t.Fatal(err)
	}
	var applied acceptanceProposalResult
	if err := json.Unmarshal(ctx.Stdout.(*bytes.Buffer).Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || applied.Task == nil {
		t.Fatalf("owner did not apply proposal: %+v", applied)
	}
}

func proposalFixture(t *testing.T) (*clikit.Ctx, *workspace.Workspace, *store.Task, store.AcceptanceProposal, *agentid.Identity, *agentid.Identity) {
	t.Helper()
	ctx, w, root := evidenceEnv(t)
	task := mkTask(t, w, "independently accepted")
	reviewer := &agentid.Identity{ID: "a-reviewer-test", Grant: model.GrantRO, Role: "reviewer"}
	runID := "run-reviewer-test"
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), procmon.Record{
		RunID: runID, Child: reviewer.ID, Task: task.ID, Role: reviewer.Role, Runtime: "codex", Started: time.Now(), Outcome: "success",
	}); err != nil {
		t.Fatal(err)
	}
	evidence, _, err := store.RunAcceptanceVerification(ctx.Cwd, reviewer.ID, "true")
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := store.NewAcceptanceProposal(w, task, reviewer, evidence, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteAcceptanceProposal(w, proposal); err != nil {
		t.Fatal(err)
	}
	return ctx, w, task, proposal, root, reviewer
}

func TestIndependentProposalSurvivesRestartAndOwnerApplies(t *testing.T) {
	ctx, w, task, proposal, root, _ := proposalFixture(t)
	reloaded, err := store.ReadAcceptanceProposal(w, task.Project, task.ID, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := applyAcceptanceProposal(ctx, w, root, task, reloaded, false, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || result.Duplicate || result.Task == nil {
		t.Fatalf("unexpected apply result: %+v", result)
	}
	closed, err := store.FindTask(w, task.ID)
	if err != nil || closed.Status != model.StatusDone {
		t.Fatalf("task not closed: status=%v err=%v", closed.Status, err)
	}
	result, err = applyAcceptanceProposal(ctx, w, root, closed, reloaded, false, true, "")
	if err != nil || !result.Duplicate {
		t.Fatalf("duplicate apply must be an idempotent success: result=%+v err=%v", result, err)
	}
}

func TestAcceptanceProposalRefusesStaleTree(t *testing.T) {
	ctx, w, task, proposal, root, _ := proposalFixture(t)
	if err := os.WriteFile(filepath.Join(w.Root, "after-review.txt"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "after-review.txt"}, {"commit", "-m", "change reviewed tree"}} {
		if out, gitErr := gitx.Run(w.Root, args...); gitErr != nil {
			t.Fatalf("git %v: %v: %s", args, gitErr, out)
		}
	}
	_, err := applyAcceptanceProposal(ctx, w, root, task, proposal, false, true, "")
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale tree must fail closed, got %v", err)
	}
}

func TestAcceptanceProposalRefusesChangedChecklist(t *testing.T) {
	ctx, w, task, proposal, root, _ := proposalFixture(t)
	if err := store.WithTask(w, task, func(fresh *store.Task) error {
		section, _ := fresh.Doc.Section("Acceptance")
		fresh.Doc.SetSection("Acceptance", section.Content+"\n- [ ] new criterion")
		return store.SaveTask(fresh)
	}); err != nil {
		t.Fatal(err)
	}
	fresh, err := store.FindTask(w, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = applyAcceptanceProposal(ctx, w, root, fresh, proposal, false, true, "")
	if err == nil || !strings.Contains(err.Error(), "checklist changed") {
		t.Fatalf("changed checklist must fail closed, got %v", err)
	}
}

func TestReadOnlyReviewerCannotApplyProposal(t *testing.T) {
	ctx, w, task, proposal, _, reviewer := proposalFixture(t)
	if reviewer.CanMutate(task.Owner()) {
		t.Fatal("read-only reviewer unexpectedly has task mutation authority")
	}
	_, err := applyAcceptanceProposal(ctx, w, reviewer, task, proposal, false, true, "")
	if err == nil || !strings.Contains(err.Error(), "not independent") {
		t.Fatalf("reviewer applying its own proposal must fail closed, got %v", err)
	}
}
