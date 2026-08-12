package ghmirror

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Task 394: a long push interrupted after its finding comments used to swallow
// the failed task closure, print the task summary, and continue without an
// honest account of which later stages were incomplete. The retry must adopt
// marker-bearing remote objects and finish without duplicating any mutation.
func TestPushInterruptionReportsIncompleteStagesAndRecoversIdempotently(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	task, err := store.CreateTask(w, "a-root", "core", "interrupted sync", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, task, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteFinding, "partial apply", store.NoteOpts{
		About: fmt.Sprintf("%d", task.Seq), Severity: "major", Body: "detail at internal/features/ghmirror/ghmirror.go:1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteDecision, "recover by marker", store.NoteOpts{
		About: fmt.Sprintf("%d", task.Seq), Rejected: "blind retries", Because: "markers converge", Body: "retry safely",
	}); err != nil {
		t.Fatal(err)
	}

	remoteIssues := []ghIssue{}
	comments := map[int][]string{}
	createdByTitle := map[string]int{}
	closed := map[int]int{}
	interrupted := true
	nextIssue := 40
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			var rows []string
			for _, issue := range remoteIssues {
				rows = append(rows, fmt.Sprintf(`{"number":%d,"title":%q,"body":%q,"state":%q}`, issue.Number, issue.Title, issue.Body, issue.State))
			}
			return "[" + strings.Join(rows, ",") + "]", nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "view":
			n, _ := strconv.Atoi(args[2])
			var rows []string
			for _, body := range comments[n] {
				rows = append(rows, fmt.Sprintf(`{"body":%q}`, body))
			}
			return `{"comments":[` + strings.Join(rows, ",") + `]}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "create":
			if interrupted && len(comments) > 0 {
				return "sync interrupted", errors.New("interrupted")
			}
			title, body := flagValue(args, "--title"), flagValue(args, "--body")
			nextIssue++
			createdByTitle[title]++
			remoteIssues = append(remoteIssues, ghIssue{Number: nextIssue, Title: title, Body: body, State: "open"})
			return fmt.Sprintf("https://github.com/owner/repo/issues/%d", nextIssue), nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "comment":
			n, _ := strconv.Atoi(args[2])
			comments[n] = append(comments[n], flagValue(args, "--body"))
			return "", nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "close":
			if interrupted {
				return "sync interrupted", errors.New("interrupted")
			}
			n, _ := strconv.Atoi(args[2])
			closed[n]++
			return "", nil
		default:
			return "", nil
		}
	}

	ctx, firstOut := releaseCtx(t, w)
	err = cmdPush(ctx, []string{"core"})
	if err == nil {
		t.Fatalf("interrupted push exited successfully:\n%s", firstOut.String())
	}
	for _, stage := range []string{"task", "closure", "decision"} {
		if !strings.Contains(strings.ToLower(err.Error()), stage) {
			t.Errorf("interruption error did not identify incomplete %s stage: %v", stage, err)
		}
	}
	if strings.Contains(firstOut.String(), "applied:") {
		t.Fatalf("failed push printed a final applied summary:\n%s", firstOut.String())
	}

	interrupted = false
	ctx, recoveredOut := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core"}); err != nil {
		t.Fatalf("recovery push: %v\n%s", err, recoveredOut.String())
	}
	if !strings.Contains(recoveredOut.String(), "applied:") {
		t.Fatalf("successful push omitted final applied summary:\n%s", recoveredOut.String())
	}
	if createdByTitle[taskIssueTitle(task)] != 1 || createdByTitle["decision: recover by marker"] != 1 {
		t.Fatalf("recovery duplicated or omitted issues: %v", createdByTitle)
	}
	if len(comments[nextIssue-1]) != 1 {
		t.Fatalf("recovery duplicated or omitted finding comment: %v", comments)
	}
	if closed[nextIssue-1] != 1 || closed[nextIssue] != 1 {
		t.Fatalf("recovery did not complete task and decision closures exactly once: %v", closed)
	}
}
