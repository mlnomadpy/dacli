package ghmirror

import (
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// A decision issue is filed CLOSED. A decision records a choice already made;
// it is not work anyone can action, and leaving it open put it in the queue
// reviewers read as "things to do". Nothing ever closed them, so they
// accumulated — this repo reached 15 open decision issues crowding out 4 real
// ones, and clearing them was a manual sweep (dacli 336).
func TestDecisionIssuesAreFiledClosed(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteDecision, "mirror decisions as closed issues", store.NoteOpts{
		Rejected: "leaving them open",
		Because:  "a record is not work",
		Body:     "the issue list should hold work",
	}); err != nil {
		t.Fatalf("create decision: %v", err)
	}

	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			return `[]`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "create":
			return "https://github.com/owner/repo/issues/77\n", nil
		default:
			return "", nil
		}
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core"}); err != nil {
		t.Fatalf("push: %v\n%s", err, out.String())
	}

	if c := findCall(calls, "issue", "create"); c == nil {
		t.Fatal("the decision issue was never created; this test would pass vacuously")
	}
	closed := findCall(calls, "issue", "close")
	if closed == nil {
		t.Fatalf("the decision issue was filed OPEN — a record is not work.\ncalls: %v", calls)
	}
	// It must close the issue it just filed, not some other number.
	sawNum := false
	for _, a := range closed {
		if a == "77" {
			sawNum = true
		}
	}
	if !sawNum {
		t.Errorf("close targeted the wrong issue: %v", closed)
	}
}

// Closing happens on CREATE, never on a re-push of an issue that already
// exists. Closing every push would fight a human who reopened a decision to
// discuss it — the mirror publishes records, it does not overrule readers.
func TestARepushDoesNotRecloseAnExistingDecisionIssue(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	notePath, err := store.CreateNote(w, "a-root", "core", model.NoteDecision, "already mirrored", store.NoteOpts{
		Rejected: "x", Because: "y", Body: "z",
	})
	if err != nil {
		t.Fatal(err)
	}
	// decisionNotes reads the id back off disk — the marker must key on the
	// same id the mirror will use, not one the test invents.
	dns, err := decisionNotes(w, "core")
	if err != nil || len(dns) != 1 {
		t.Fatalf("decisionNotes: %v (%d notes) for %s", err, len(dns), notePath)
	}
	noteID := dns[0].id

	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			// The remote already carries this decision, adopted by marker, and
			// somebody has REOPENED it.
			return `[{"number":88,"title":"decision: already mirrored","body":"` +
				decisionMarker(w, noteID) + `","state":"open"}]`, nil
		default:
			return "", nil
		}
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core"}); err != nil {
		t.Fatalf("push: %v\n%s", err, out.String())
	}

	if findCall(calls, "issue", "create") != nil {
		t.Fatal("a re-push created a duplicate decision issue")
	}
	if c := findCall(calls, "issue", "close"); c != nil {
		t.Errorf("a re-push closed an existing decision issue %v — that overrules a human who reopened it", c)
	}
}
