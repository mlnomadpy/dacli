package vcs

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// prEnv builds a workspace with one project and one task carrying acceptance
// criteria, and returns both. DACLI_AGENT is cleared so the acting identity is
// root regardless of who runs the suite.
func prEnv(t *testing.T) (*workspace.Workspace, *store.Task) {
	t.Helper()
	t.Setenv("DACLI_AGENT", "")
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	tk, err := store.CreateTask(w, "a-root", "p", "Enrich PR", store.TaskOpts{Accept: []string{"body carries findings", "Fixes line present"}})
	if err != nil {
		t.Fatal(err)
	}
	return w, tk
}

func TestPRBodyCarriesAcceptanceFindingsAndFixes(t *testing.T) {
	w, tk := prEnv(t)

	// A finding about this task, plus one about a different task that must NOT
	// leak into the body.
	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "race in the merge path",
		store.NoteOpts{About: tk.ID, Severity: "major", Body: "double free at lifecycle.go:200"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "unrelated finding",
		store.NoteOpts{About: "t-someone-else", Severity: "minor", Body: "not about this task"}); err != nil {
		t.Fatal(err)
	}

	// Link the task to a mirrored issue via its own github: frontmatter block,
	// exactly as ghmirror writes it at push.
	tk.Doc.Front.SetBlock("github", "  issue: 42\n  repo: acme/widgets")

	body := prBody(w, tk)

	if !strings.Contains(body, "### Acceptance") {
		t.Errorf("body missing acceptance section:\n%s", body)
	}
	if !strings.Contains(body, "body carries findings") {
		t.Errorf("body missing an acceptance criterion:\n%s", body)
	}
	if !strings.Contains(body, "Fixes #42") {
		t.Errorf("body missing Fixes line:\n%s", body)
	}
	if !strings.Contains(body, "### Findings") || !strings.Contains(body, "double free at lifecycle.go:200") {
		t.Errorf("body missing the task's finding:\n%s", body)
	}
	if !strings.Contains(body, "**major**") {
		t.Errorf("body missing the finding severity tag:\n%s", body)
	}
	if strings.Contains(body, "not about this task") {
		t.Errorf("body leaked a finding about a different task:\n%s", body)
	}
}

func TestPRBodySkipsFixesWhenUnlinked(t *testing.T) {
	w, tk := prEnv(t)
	body := prBody(w, tk)
	if strings.Contains(body, "Fixes #") {
		t.Errorf("unlinked task should carry no Fixes line:\n%s", body)
	}
	// Acceptance still renders even with no findings and no issue link.
	if !strings.Contains(body, "### Acceptance") {
		t.Errorf("body missing acceptance section:\n%s", body)
	}
	if strings.Contains(body, "### Findings") {
		t.Errorf("body should have no Findings section when none are filed:\n%s", body)
	}
}

func TestTaskFixesLineIgnoresMalformedIssue(t *testing.T) {
	_, tk := prEnv(t)
	// A github block with no issue key (repo-only) must not fabricate a Fixes line.
	tk.Doc.Front.SetBlock("github", "  repo: acme/widgets")
	if got := taskFixesLine(tk); got != "" {
		t.Errorf("expected no Fixes line for a block without an issue, got %q", got)
	}
	tk.Doc.Front.SetBlock("github", "  issue: 7\n  repo: acme/widgets")
	if got := taskFixesLine(tk); got != "Fixes #7" {
		t.Errorf("expected Fixes #7, got %q", got)
	}
}

func TestVerdictReviewRendersRecordedVerdicts(t *testing.T) {
	w, tk := prEnv(t)

	// Mirror the verify-verdict: convention verify writes (execution.VerdictRecord).
	// The vcs slice must not import execution, so the contract is the string, not
	// the function — exercise the reader against the exact shape verify emits.
	if _, err := eventlog.Append(w, "a-seat1", model.EventComment, tk.ID, "",
		"verify-verdict: confirmed — claude-code (a-seat1) on claim: race in the merge path — reproduces under -race"); err != nil {
		t.Fatal(err)
	}
	if _, err := eventlog.Append(w, "a-seat2", model.EventComment, tk.ID, "",
		"verify-verdict: refuted — gemini (a-seat2) on claim: race in the merge path — cannot reproduce"); err != nil {
		t.Fatal(err)
	}
	// A plain comment (not a verdict) must be ignored.
	if _, err := eventlog.Append(w, "a-other", model.EventComment, tk.ID, "", "just chatting"); err != nil {
		t.Fatal(err)
	}

	review := verdictReview(w, tk)
	if !strings.Contains(review, "dacli verify panel") {
		t.Errorf("review missing header:\n%s", review)
	}
	if !strings.Contains(review, "confirmed — claude-code") || !strings.Contains(review, "refuted — gemini") {
		t.Errorf("review missing a verdict line:\n%s", review)
	}
	if strings.Contains(review, "verify-verdict:") {
		t.Errorf("review should strip the marker prefix:\n%s", review)
	}
	if strings.Contains(review, "just chatting") {
		t.Errorf("review leaked a non-verdict comment:\n%s", review)
	}
	// Chronological order: seat1 voted before seat2.
	if strings.Index(review, "a-seat1") > strings.Index(review, "a-seat2") {
		t.Errorf("verdicts not in chronological order:\n%s", review)
	}
}

func TestVerdictReviewEmptyWhenNoVerdicts(t *testing.T) {
	w, tk := prEnv(t)
	if got := verdictReview(w, tk); got != "" {
		t.Errorf("expected empty review with no recorded verdicts, got %q", got)
	}
}
