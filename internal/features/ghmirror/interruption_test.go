package ghmirror

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestPushAdoptsCreateWhoseOutputWasInterruptedBeforeMapping(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	task, err := store.CreateTask(w, "a-root", "core", "recover exact create", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	var remote []ghIssue
	creates := 0
	interrupt := true
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			if len(remote) == 0 {
				return `[]`, nil
			}
			return fmt.Sprintf(`[{"number":71,"title":%q,"body":%q,"state":"open"}]`, remote[0].Title, remote[0].Body), nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "create":
			creates++
			remote = append(remote, ghIssue{Number: 71, Title: flagValue(args, "--title"), Body: flagValue(args, "--body")})
			if interrupt {
				return "managed output interrupted", errors.New("interrupted")
			}
			return "https://github.com/owner/repo/issues/71", nil
		default:
			return "", nil
		}
	}

	ctx, _ := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core"}); err == nil {
		t.Fatal("first push should report the interrupted create")
	}
	interrupt = false
	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core"}); err != nil {
		t.Fatalf("recovery push: %v\n%s", err, out.String())
	}
	if creates != 1 {
		t.Fatalf("recovery issued %d creates, want exactly the interrupted create", creates)
	}
	reloaded, err := store.FindTask(w, task.ID)
	if err != nil || mappedIssue(reloaded) != 71 {
		t.Fatalf("recovery did not adopt issue #71: task=%+v err=%v", reloaded, err)
	}
}

func TestConcurrentPushesSerializeMarkerCheckAndCreate(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	if _, err := store.CreateTask(w, "a-root", "core", "one remote issue", store.TaskOpts{}); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var remote []ghIssue
	creates := 0
	listCalls := 0
	bothListed := make(chan struct{})
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			mu.Lock()
			listCalls++
			if listCalls == 2 {
				close(bothListed)
			}
			if len(remote) == 0 {
				mu.Unlock()
				select {
				case <-bothListed:
				case <-time.After(100 * time.Millisecond):
				}
				return `[]`, nil
			}
			issue := remote[0]
			mu.Unlock()
			return fmt.Sprintf(`[{"number":81,"title":%q,"body":%q,"state":"open"}]`, issue.Title, issue.Body), nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "create":
			mu.Lock()
			defer mu.Unlock()
			creates++
			remote = append(remote, ghIssue{Number: 81, Title: flagValue(args, "--title"), Body: flagValue(args, "--body")})
			return "https://github.com/owner/repo/issues/81", nil
		default:
			return "", nil
		}
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			ctx, _ := releaseCtx(t, w)
			errs <- cmdPush(ctx, []string{"core"})
		}()
	}
	close(start)
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent push %d: %v", i, err)
		}
	}
	if creates != 1 {
		t.Fatalf("concurrent pushes created %d issues, want 1", creates)
	}
}

func TestPushReportsAllDuplicateMarkersAndChoosesLowest(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	task, err := store.CreateTask(w, "a-root", "core", "duplicate marker", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	mk := marker(w, task)
	creates := 0
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			return fmt.Sprintf(`[{"number":29,"title":"later","body":%q},{"number":28,"title":"first","body":%q}]`, mk, mk), nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "create":
			creates++
		}
		return "", nil
	}
	ctx, dryOut := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatalf("dry-run: %v\n%s", err, dryOut.String())
	}
	unmapped, _ := store.FindTask(w, task.ID)
	if mappedIssue(unmapped) != 0 || !strings.Contains(dryOut.String(), "#28, #29") {
		t.Fatalf("dry-run did not report ambiguity read-only: mapped=%d\n%s", mappedIssue(unmapped), dryOut.String())
	}
	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core"}); err != nil {
		t.Fatalf("push: %v\n%s", err, out.String())
	}
	if creates != 0 || !strings.Contains(out.String(), "#28, #29") || !strings.Contains(out.String(), "canonical mapping is #28") {
		t.Fatalf("duplicate condition not reconciled explicitly (creates=%d):\n%s", creates, out.String())
	}
	reloaded, _ := store.FindTask(w, task.ID)
	if mappedIssue(reloaded) != 28 {
		t.Fatalf("mapped issue = %d, want deterministic #28", mappedIssue(reloaded))
	}
}

func TestPushLockReleasesAndRecoversDeadOwnerWhileDryRunTakesNoLease(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	lock := githubPushLockPath(w, "owner/repo")
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		}
		if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
			return `[]`, nil
		}
		return "", nil
	}
	ctx, _ := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(lock)); !os.IsNotExist(err) {
		t.Fatalf("dry-run created mutating lock state: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	host, _ := os.Hostname()
	dead := fmt.Sprintf("{\"pid\":99999999,\"pid_start\":\"dead\",\"host\":%q,\"token\":\"dead-owner\",\"ts\":%q}\n", host, time.Now().Add(-10*time.Second).UTC().Format(time.RFC3339Nano))
	if err := os.WriteFile(lock, []byte(dead), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, _ = releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core"}); err != nil {
		t.Fatalf("push did not recover dead lock owner: %v", err)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatalf("successful push did not release lock: %v", err)
	}
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		}
		if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
			return "truncated transport", errors.New("connection reset")
		}
		return "", nil
	}
	ctx, _ = releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core"}); err == nil {
		t.Fatal("failed marker read unexpectedly succeeded")
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatalf("failed push did not release lock: %v", err)
	}
}

func TestPushLockScopeIsWorkspaceAndRepository(t *testing.T) {
	w1 := &workspace.Workspace{Root: t.TempDir()}
	w2 := &workspace.Workspace{Root: t.TempDir()}
	a := githubPushLockPath(w1, "Owner/Repo")
	if a != githubPushLockPath(w1, "owner/repo") {
		t.Fatal("repository identity casing produced two locks")
	}
	if a == githubPushLockPath(w1, "owner/other") {
		t.Fatal("different linked repositories share one lock")
	}
	if a == githubPushLockPath(w2, "owner/repo") {
		t.Fatal("different workspaces share one lock")
	}
}
