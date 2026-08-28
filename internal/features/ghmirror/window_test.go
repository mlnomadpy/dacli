package ghmirror

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// task 275: `github push` must mirror only the requested window (explicit refs
// and/or a --since cutoff), not the whole backlog. These tests pin the window
// selection, the marker-less title adoption, and the end-to-end property that a
// done set far larger than the window still files just the windowed task.

// windowTask builds an in-memory done task carrying a seq, title, and created
// stamp — enough for selectTaskWindow, which reads only the ref forms
// (seq/slug/id) and the `created` frontmatter.
func windowTask(seq int, title, created string) *store.Task {
	d := &mdstore.Doc{Sections: []mdstore.Section{{Level: 1, Title: title}}}
	if created != "" {
		d.Front.Set("created", created)
	}
	return &store.Task{
		ID:     fmt.Sprintf("t-%03d", seq),
		Seq:    seq,
		Slug:   store.Slugify(title),
		Title:  title,
		Status: model.StatusDone,
		Doc:    d,
	}
}

// The acceptance case: a workspace whose done set is far larger than the window.
// Given 300 done tasks and a single explicit ref, the window is exactly that one
// task — the whole point of task 275 (a mature project mirrors one wave, not the
// entire backlog).
func TestSelectTaskWindowMirrorsOnlyExplicitRefInALargeDoneSet(t *testing.T) {
	var tasks []*store.Task
	for i := 1; i <= 300; i++ {
		tasks = append(tasks, windowTask(i, fmt.Sprintf("done task %d", i), ""))
	}
	got, err := selectTaskWindow(tasks, []string{"275"}, time.Time{})
	if err != nil {
		t.Fatalf("selectTaskWindow: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("window returned %d tasks, want 1 out of %d done tasks", len(got), len(tasks))
	}
	if got[0].Seq != 275 {
		t.Fatalf("window mirrored task %d, want 275", got[0].Seq)
	}
}

// --since selects the recent tasks by created stamp; combining it with an
// explicit ref is a UNION — the named (possibly old) task PLUS everything since
// the cutoff.
func TestSelectTaskWindowSinceAndUnion(t *testing.T) {
	now := time.Now()
	stamp := func(d time.Duration) string { return now.Add(d).UTC().Format(time.RFC3339) }
	old := windowTask(1, "old", stamp(-72*time.Hour))
	recentA := windowTask(2, "recent a", stamp(-30*time.Minute))
	recentB := windowTask(3, "recent b", stamp(-10*time.Minute))
	tasks := []*store.Task{old, recentA, recentB}

	since := now.Add(-1 * time.Hour)
	got, err := selectTaskWindow(tasks, nil, since)
	if err != nil {
		t.Fatalf("since window: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("--since 1h returned %d tasks, want 2 (the recent pair, not the 72h-old one)", len(got))
	}

	// Ref to the OLD task PLUS the since window → all three, and the old task is
	// counted once (ref inclusion short-circuits the since check).
	got, err = selectTaskWindow(tasks, []string{"1"}, since)
	if err != nil {
		t.Fatalf("union window: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("union (ref to old + --since) returned %d tasks, want 3", len(got))
	}
}

// The default push — no refs, no since — mirrors the whole backlog exactly as
// before, so the window is purely additive and does not change existing behavior.
func TestSelectTaskWindowEmptyReturnsAll(t *testing.T) {
	tasks := []*store.Task{windowTask(1, "a", ""), windowTask(2, "b", "")}
	got, err := selectTaskWindow(tasks, nil, time.Time{})
	if err != nil {
		t.Fatalf("selectTaskWindow: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("empty window returned %d tasks, want all 2", len(got))
	}
}

// An explicit ref that matches no task is a NOT-FOUND error (exit 4), never a
// silent empty window — an operator who asked to mirror a task must hear that it
// was not found, not watch push report success having filed nothing for it.
func TestSelectTaskWindowUnknownRefIsNotFound(t *testing.T) {
	tasks := []*store.Task{windowTask(1, "a", "")}
	_, err := selectTaskWindow(tasks, []string{"999"}, time.Time{})
	if err == nil {
		t.Fatalf("an unmatched ref must error, not silently mirror nothing")
	}
	if code := clikit.ExitCode(err); code != 4 {
		t.Fatalf("unknown-ref exit = %d, want 4 (not found)", code)
	}
}

// A --since window excludes a task with no parseable created stamp: "since"
// means demonstrably created after the cutoff, never "assume recent".
func TestSelectTaskWindowSinceExcludesUndatedTask(t *testing.T) {
	got, err := selectTaskWindow([]*store.Task{windowTask(1, "undated", "")}, nil, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("selectTaskWindow: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a task with no created stamp must fall OUTSIDE a --since window, got %d", len(got))
	}
}

// task 275 adoption: a hand-filed issue carries the canonical `NNN: <title>` but
// no dacli marker, so the marker search misses it — findByTitle adopts it by
// exact title rather than filing a duplicate.
func TestFindByTitleAdoptsUnmarkedIssue(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		return `[{"number":42,"title":"275: mirror one wave","body":"filed by hand, no marker"}]`, nil
	}
	idx := newMarkerIndex(w, "owner/repo")

	if got := idx.find("<!-- dacli:t-abc ws:w1 -->"); got != 0 {
		t.Fatalf("marker find must miss an unmarked issue, got %d", got)
	}
	if got := idx.findByTitle("275: mirror one wave"); got != 42 {
		t.Fatalf("findByTitle = %d, want 42 (adopt the hand-filed issue)", got)
	}
	// A partial/prefix title must NOT cross-adopt — the match is the full title.
	if got := idx.findByTitle("275: mirror"); got != 0 {
		t.Fatalf("a partial title must not adopt, got %d", got)
	}
	if got := idx.findByTitle(""); got != 0 {
		t.Fatalf("an empty title must never adopt, got %d", got)
	}
}

// On the off chance the repo already holds two identically-titled issues, the
// lowest number wins — a deterministic tie-break, so a re-push converges on the
// same one instead of oscillating.
func TestFindByTitlePicksLowestOnCollision(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		return `[{"number":9,"title":"007: dup","body":"a"},{"number":4,"title":"007: dup","body":"b"}]`, nil
	}
	idx := newMarkerIndex(w, "owner/repo")
	if got := idx.findByTitle("007: dup"); got != 4 {
		t.Fatalf("findByTitle on a title collision = %d, want the lowest number 4", got)
	}
}

// flagValue returns the value following flag in an argv, or "".
func flagValue(args []string, flag string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

// End-to-end: a workspace with a done set far larger than the window still files
// EXACTLY the windowed task and nothing else — the property task 275 exists to
// guarantee. Without the window, all 20 done tasks would create/adopt an issue.
func TestPushMirrorsOnlyWindowedTaskInLargeDoneSet(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")

	var target *store.Task
	for i := 0; i < 20; i++ {
		nt, err := store.CreateTask(w, "a-root", "core", fmt.Sprintf("backlog item %d", i), store.TaskOpts{})
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		if err := store.MoveTask(w, nt, model.StatusDone); err != nil {
			t.Fatalf("move done: %v", err)
		}
		if i == 7 {
			target = nt
		}
	}

	var created []string // the title of every `issue create` the push issued
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			return "[]", nil // empty repo — the windowed task is a fresh create
		case len(args) >= 1 && args[0] == "api":
			return "", nil // milestone list/POST never confirms, so no --milestone
		case len(args) >= 2 && args[0] == "issue" && args[1] == "create":
			created = append(created, flagValue(args, "--title"))
			return "https://github.com/owner/repo/issues/500", nil
		default:
			return "", nil // labels, edits, close — best-effort, ignored here
		}
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core", fmt.Sprintf("%d", target.Seq)}); err != nil {
		t.Fatalf("push: %v\n%s", err, out.String())
	}

	if len(created) != 1 {
		t.Fatalf("push filed %d issue(s) for a 1-task window over 20 done tasks; want exactly 1: %v", len(created), created)
	}
	if want := taskIssueTitle(target); created[0] != want {
		t.Fatalf("push filed %q, want the windowed task %q", created[0], want)
	}
}

func TestPushClosureOnlyPublishesNoFindingsOrDecisions(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	tk, err := store.CreateTask(w, "a-root", "core", "landed task", store.TaskOpts{Accept: []string{"verified"}})
	if err != nil {
		t.Fatal(err)
	}
	tk.Doc.Front.SetBlock("github", "  issue: 841\n  repo: owner/repo")
	if err := store.SaveTask(tk); err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tk, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteFinding, "private operational evidence", store.NoteOpts{About: tk.ID, Body: "must not be published by closure-only"}); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, append([]string(nil), args...))
		if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		}
		return "", nil
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core", tk.ID, "--closure-only"}); err != nil {
		t.Fatalf("closure-only: %v\n%s", err, out.String())
	}
	closed := findCall(calls, "issue", "close")
	if closed == nil || !strings.Contains(strings.Join(closed, " "), "841") {
		t.Fatalf("mapped issue was not closed: %v", calls)
	}
	for _, forbidden := range [][]string{{"issue", "comment"}, {"issue", "create"}, {"issue", "edit"}, {"issue", "list"}, {"label", "create"}} {
		if got := findCall(calls, forbidden...); got != nil {
			t.Fatalf("closure-only leaked into %v: %v", forbidden, got)
		}
	}
	if !strings.Contains(out.String(), "no findings or decisions published") {
		t.Fatalf("closure-only output omitted disclosure boundary:\n%s", out.String())
	}
}

// task 298: the window scoped the TASKS but the decision and finding-issue mirrors
// rode along unscoped — a decision about a task OUTSIDE the one-task window still
// filed a public issue. On a public repo that is an unbounded disclosure a scoped
// push must not do. This pins the property that a windowed push creates NO issue
// for any object (task, decision, or finding) outside the window, and that the
// blast-radius plan line states the create counts before any issue is filed.
func TestPushWindowScopesDecisionsAndFindingsToTheWindow(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")

	mkDone := func(title string) *store.Task {
		nt, err := store.CreateTask(w, "a-root", "core", title, store.TaskOpts{})
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		if err := store.MoveTask(w, nt, model.StatusDone); err != nil {
			t.Fatalf("move done: %v", err)
		}
		return nt
	}
	inWin := mkDone("inside the window")
	outWin := mkDone("outside the window")

	// A decision AND a finding about each task, scoped by the task's seq ref.
	mustNote := func(kind model.NoteKind, title string, about *store.Task) {
		if _, err := store.CreateNote(w, "a-root", "core", kind, title, store.NoteOpts{
			About:    fmt.Sprintf("%d", about.Seq),
			Rejected: "the rejected alternative",
			Because:  "the recorded reason",
			Severity: "minor",
			Body:     "detail at internal/x.go:1",
		}); err != nil {
			t.Fatalf("create %s note: %v", kind, err)
		}
	}
	mustNote(model.NoteDecision, "decide inside", inWin)
	mustNote(model.NoteDecision, "decide outside", outWin)
	mustNote(model.NoteFinding, "find inside", inWin)
	mustNote(model.NoteFinding, "find outside", outWin)

	var created []string // the --title of every `issue create` the push issued
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			return "[]", nil // empty repo — every in-window object is a fresh create
		case len(args) >= 1 && args[0] == "api":
			return "", nil // milestone never confirms
		case len(args) >= 2 && args[0] == "issue" && args[1] == "create":
			created = append(created, flagValue(args, "--title"))
			return "https://github.com/owner/repo/issues/500", nil
		default:
			return "", nil // labels, edits, close — best-effort, ignored here
		}
	}

	// --findings-as-issues --with-tasks exercises all three mirrors (tasks,
	// decisions, standalone finding issues) on a single one-task window.
	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core", fmt.Sprintf("%d", inWin.Seq), "--findings-as-issues", "--with-tasks"}); err != nil {
		t.Fatalf("push: %v\n%s", err, out.String())
	}

	has := func(title string) bool {
		for _, c := range created {
			if c == title {
				return true
			}
		}
		return false
	}
	// The in-window objects ARE published.
	if !has(taskIssueTitle(inWin)) {
		t.Errorf("windowed task issue not created: %v", created)
	}
	if !has("decision: decide inside") {
		t.Errorf("in-window decision not created: %v", created)
	}
	if !has("find inside") {
		t.Errorf("in-window finding issue not created: %v", created)
	}
	// The out-of-window objects create NO issue — the property task 298 exists for.
	if has(taskIssueTitle(outWin)) {
		t.Errorf("out-of-window task issue leaked: %v", created)
	}
	if has("decision: decide outside") {
		t.Errorf("out-of-window decision rode along a scoped push onto the repo: %v", created)
	}
	if has("find outside") {
		t.Errorf("out-of-window finding issue rode along a scoped push onto the repo: %v", created)
	}

	// The blast radius is stated before any issue is created.
	if !strings.Contains(out.String(), "will create 1 task, 1 decision, 1 finding issue(s)") {
		t.Errorf("plan line missing/incorrect blast radius:\n%s", out.String())
	}
}
