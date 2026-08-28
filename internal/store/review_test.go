package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func validReview(verdict ReviewVerdict, commit, tree string) IndependentReviewResult {
	r := IndependentReviewResult{Schema: ReviewResultSchema, Verdict: verdict, ReviewerID: "a-reviewer-1", ReviewerRole: "reviewer", Runtime: "codex", Model: "gpt", Grant: "ro", IndependentOf: []string{"a-implementer-1"}, CommitSHA: commit, TreeSHA: tree, ObservedPRGeneration: 2, ObservedAt: time.Unix(10, 0)}
	if verdict == ReviewRequestChanges {
		r.Findings = []ReviewFinding{{ID: "REV-001", Severity: "major", File: "internal/x.go", Line: 7, EndLine: 8, Evidence: "/private/workspace/internal/x.go contains secret detail", AffectedInvariant: "failed output must not count as approval", SuggestedVerification: "go test ./internal/store -run Review"}}
	}
	return r
}

func TestIndependentReviewVerdictsFailClosed(t *testing.T) {
	for _, verdict := range []ReviewVerdict{ReviewInconclusive, ReviewNoResponse, ReviewInfrastructureFailure} {
		t.Run(string(verdict), func(t *testing.T) {
			tx := ReviewTransaction{State: ReviewAwaiting, MaxCorrections: 2}
			if err := tx.Apply(validReview(verdict, "c1", "t1"), "c1", "t1", time.Now()); err == nil {
				t.Fatalf("%s counted as success", verdict)
			}
			if tx.State != ReviewHalted {
				t.Fatalf("state=%s, want halted", tx.State)
			}
		})
	}
}

func TestReviewTransactionBindsCorrectionAndRereviewToFreshTree(t *testing.T) {
	tx := ReviewTransaction{State: ReviewAwaiting, MaxCorrections: 1}
	if err := tx.Apply(validReview(ReviewRequestChanges, "c1", "t1"), "c1", "t1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if tx.State != ReviewCorrection || tx.CorrectionTurns != 1 || len(tx.FindingIDs) != 1 || tx.FindingIDs[0] != "REV-001" {
		t.Fatalf("after finding: %+v", tx)
	}
	if err := tx.MarkCorrected("c2", "t2", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Apply(validReview(ReviewApprove, "c1", "t1"), "c2", "t2", time.Now()); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale approval=%v, want stale refusal", err)
	}

	tx.State = ReviewAwaitingRereview
	if err := tx.Apply(validReview(ReviewApprove, "c2", "t2"), "c2", "t2", time.Now()); err != nil {
		t.Fatal(err)
	}
	if tx.State != ReviewApproved {
		t.Fatalf("state=%s, want approved", tx.State)
	}
}

func TestReviewTransactionRefusesSkippedRereviewAndExtraCorrection(t *testing.T) {
	tx := ReviewTransaction{State: ReviewAwaiting, MaxCorrections: 1}
	if err := tx.Apply(validReview(ReviewRequestChanges, "c1", "t1"), "c1", "t1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if tx.State == ReviewApproved {
		t.Fatal("request-changes skipped re-review")
	}
	if err := tx.Apply(validReview(ReviewRequestChanges, "c1", "t1"), "c1", "t1", time.Now()); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("extra correction=%v, want bounded refusal", err)
	}
}

func TestReviewProjectionDoesNotLeakPrivateEvidence(t *testing.T) {
	r := validReview(ReviewRequestChanges, "c1", "t1")
	p, err := r.PublicProjection()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(p)
	got := string(raw)
	for _, forbidden := range []string{"/private/workspace", "a-reviewer", "a-implementer", "codex", "gpt", "secret detail"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("public projection leaked %q: %s", forbidden, got)
		}
	}
	for _, want := range []string{"REV-001", "internal/x.go", "failed output must not count as approval", "go test"} {
		if !strings.Contains(got, want) {
			t.Fatalf("public projection omitted %q: %s", want, got)
		}
	}
}

func TestReviewTransactionCrashRestartRoundTrip(t *testing.T) {
	root := t.TempDir()
	w, err := workspace.Init(root, "review")
	if err != nil {
		t.Fatal(err)
	}
	tx := ReviewTransaction{Project: "p", TaskID: "t-review", Branch: "dacli/1-review", State: ReviewCorrection, CorrectionTurns: 1, MaxCorrections: 2, PriorCommit: "c1", PriorTree: "t1", FindingIDs: []string{"REV-001"}, UpdatedAt: time.Now().UTC()}
	if err := WriteReviewTransaction(w, tx); err != nil {
		t.Fatal(err)
	}
	got, err := ReadReviewTransaction(w, "p", "t-review")
	if err != nil {
		t.Fatal(err)
	}
	if got.State != ReviewCorrection || got.CorrectionTurns != 1 || got.PriorTree != "t1" || got.FindingIDs[0] != "REV-001" {
		t.Fatalf("restart lost transaction: %+v", got)
	}
	if err := os.WriteFile(ReviewTransactionPath(w, "p", "t-review"), []byte(`{"schema":"wrong"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadReviewTransaction(w, "p", "t-review"); err == nil {
		t.Fatal("corrupt checkpoint treated as empty/success")
	}
	if !strings.HasPrefix(ReviewTransactionPath(w, "p", "t-review"), filepath.Join(root, workspace.Dir)) {
		t.Fatal("journal escaped workspace")
	}
}

func TestReviewResultRequiresROIdentityAndCompleteFindings(t *testing.T) {
	r := validReview(ReviewRequestChanges, "c1", "t1")
	r.Grant = "rw"
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "ro") {
		t.Fatalf("rw reviewer accepted: %v", err)
	}
	r = validReview(ReviewRequestChanges, "c1", "t1")
	r.Findings[0].SuggestedVerification = ""
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "suggested verification") {
		t.Fatalf("incomplete finding accepted: %v", err)
	}
}

func TestParseReviewOutputUsesTerminalStructuredEnvelope(t *testing.T) {
	r := validReview(ReviewApprove, "c1", "t1")
	raw, _ := json.Marshal(r)
	got, err := ParseReviewOutput("provider chatter\n" + ReviewOutputMarker + string(raw) + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != ReviewApprove || got.TreeSHA != "t1" {
		t.Fatalf("parsed=%+v", got)
	}
	if _, err := ParseReviewOutput("provider exited successfully with no contract"); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing envelope=%v, want failure", err)
	}
}
