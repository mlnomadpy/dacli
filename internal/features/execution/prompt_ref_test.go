package execution

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

// Generated worker commands cross the project boundary through the shared
// workspace resolver. They must carry the stable task identity even when the
// sequence shown to humans is ambiguous (issue #636).
func TestPromptSuffixUsesStableTaskIDForMutatingCommands(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "assigned task", store.TaskOpts{Accept: []string{"done"}})
	if _, err := store.CreateProject(w, agentid.RootID, "Other", "other", "", ""); err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateTask(w, agentid.RootID, "other", "colliding task", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	if other.Seq != task.Seq {
		t.Fatalf("fixture must collide on sequence: got %d and %d", task.Seq, other.Seq)
	}

	f, err := clikit.ParseFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := promptSuffix(w, f, task, "a-child", model.GrantRW)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{
		"task check " + task.ID,
		"task done " + task.ID,
		"accept " + task.ID,
		"commit \"" + task.ID + ": <what changed>\" --task " + task.ID,
	} {
		if !strings.Contains(out, command) {
			t.Errorf("generated instructions missing stable command %q", command)
		}
	}
	numeric := fmt.Sprintf("%03d", task.Seq)
	for _, command := range []string{"task check " + numeric, "task done " + numeric, "accept " + numeric} {
		if strings.Contains(out, command) {
			t.Errorf("generated instructions contain ambiguous command %q", command)
		}
	}
}

func TestReviewPromptUsesTaskBranchWithoutGitHubWhenPRFirstIsOff(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "Review local branch", store.TaskOpts{Accept: []string{"done"}})
	f, err := clikit.ParseFlags([]string{"--review"})
	if err != nil {
		t.Fatal(err)
	}

	out, err := promptSuffix(w, f, task, "a-reviewer", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	branch := taskBranch(task)
	for _, want := range []string{
		"local landing mode",
		"git diff main..." + branch,
		"git show " + branch,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("local review prompt missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "gh pr ") {
		t.Errorf("local review prompt must not require gh:\n%s", out)
	}
}

func TestReviewPromptUsesRepositoryTrunkInsteadOfHardcodedMain(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	if out, err := gitx.Run(w.Root, "branch", "-m", "master"); err != nil {
		t.Fatalf("rename default branch: %v\n%s", err, out)
	}
	task := mustTask(t, w, "Review master branch", store.TaskOpts{Accept: []string{"done"}})
	f, err := clikit.ParseFlags([]string{"--review"})
	if err != nil {
		t.Fatal(err)
	}

	out, err := promptSuffix(w, f, task, "a-reviewer", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	want := "git diff master..." + taskBranch(task)
	if !strings.Contains(out, want) {
		t.Fatalf("local review prompt missing resolved trunk %q:\n%s", want, out)
	}
}

func TestReviewPromptUsesGitHubPRWhenPRFirstIsOn(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "Review PR", store.TaskOpts{Accept: []string{"done"}})
	f, err := clikit.ParseFlags([]string{"--review", "--pr", "--pr-number", "42"})
	if err != nil {
		t.Fatal(err)
	}

	out, err := promptSuffix(w, f, task, "a-reviewer", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"gh pr view 42", "gh pr diff <number>", "gh pr review <number>"} {
		if !strings.Contains(out, want) {
			t.Errorf("PR review prompt missing %q:\n%s", want, out)
		}
	}
}
