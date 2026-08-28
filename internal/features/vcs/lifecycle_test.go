package vcs

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/publication"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// prEnv builds a workspace with one project and one task carrying acceptance
// criteria, and returns both. DACLI_AGENT is cleared so the acting identity is
// root regardless of who runs the suite.
func prEnv(t *testing.T) (*workspace.Workspace, *store.Task) {
	t.Helper()
	unsetAgentEnv(t)
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
	if err := store.MoveTask(w, tk, model.StatusDone); err != nil {
		t.Fatal(err)
	}

	body := prBody(w, tk, false)

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
	if strings.Contains(body, "TRUST GRADE") {
		t.Errorf("body without --with-verdicts must carry no trust-grade section:\n%s", body)
	}
}

func TestPRBodySkipsFixesWhenUnlinked(t *testing.T) {
	w, tk := prEnv(t)
	body := prBody(w, tk, false)
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

// TestPRBodyWithVerdictsRendersTrustGradeAndTally covers task 146's acceptance:
// --with-verdicts must put a loud trust-grade summary + per-finding verdict
// tally into the PR BODY as a first-class section, ahead of Acceptance/Findings,
// using only the trust: frontmatter verify already stamps and the
// verify-verdict: comment events verify already records (no new collection).
func TestPRBodyWithVerdictsRendersTrustGradeAndTally(t *testing.T) {
	w, tk := prEnv(t)

	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "race in the merge path",
		store.NoteOpts{About: tk.ID, Severity: "major", Body: "double free at lifecycle.go:200"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GradeFinding(w, "p", "race in the merge path", "confirmed"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "unchecked lead",
		store.NoteOpts{About: tk.ID, Severity: "minor", Body: "smells wrong but nobody has verified it"}); err != nil {
		t.Fatal(err)
	}

	// Recorded panel votes for the confirmed finding's claim.
	if _, err := eventlog.Append(w, "a-seat1", model.EventComment, tk.ID, "",
		"verify-verdict: confirmed — claude-code (a-seat1) on claim: race in the merge path — reproduces under -race"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := eventlog.Append(w, "a-seat2", model.EventComment, tk.ID, "",
		"verify-verdict: refuted — gemini (a-seat2) on claim: race in the merge path — cannot reproduce"); err != nil {
		t.Fatal(err)
	}

	withoutVerdicts := prBody(w, tk, false)
	if strings.Contains(withoutVerdicts, "TRUST GRADE") {
		t.Errorf("plain `dacli pr` (no --with-verdicts) must render no trust-grade section:\n%s", withoutVerdicts)
	}

	body := prBody(w, tk, true)
	if !strings.Contains(body, "TRUST GRADE") {
		t.Fatalf("--with-verdicts body missing a loud trust-grade section:\n%s", body)
	}
	if !strings.Contains(body, "1 confirmed") || !strings.Contains(body, "1 unverified") {
		t.Errorf("body missing the aggregate trust tally:\n%s", body)
	}
	if !strings.Contains(body, "race in the merge path") || !strings.Contains(body, "1 confirmed, 1 refuted") {
		t.Errorf("body missing the per-finding panel vote tally:\n%s", body)
	}
	if !strings.Contains(body, "unchecked lead") || !strings.Contains(body, "no panel votes recorded") {
		t.Errorf("body missing the ungraded finding's tally row:\n%s", body)
	}
	// LOUD and first-class: the trust-grade section must appear before
	// Acceptance/Findings, not buried after them.
	if i, j := strings.Index(body, "TRUST GRADE"), strings.Index(body, "### Acceptance"); i < 0 || j < 0 || i > j {
		t.Errorf("trust-grade section must lead the body, ahead of Acceptance:\n%s", body)
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
	tk.Status = model.StatusDone
	if got := taskFixesLine(tk); got != "Fixes #7" {
		t.Errorf("expected Fixes #7, got %q", got)
	}
}

func TestTaskFixesLineLeavesNonterminalIssueOpen(t *testing.T) {
	_, tk := prEnv(t)
	tk.Doc.Front.SetBlock("github", "  issue: 841\n  repo: acme/widgets")
	if got := taskFixesLine(tk); got != "" {
		t.Fatalf("nonterminal PR would close issue before post-landing verification: %q", got)
	}
}

func TestPublicPRProjectionWithholdsPrivateEvidenceAndUsesRefs(t *testing.T) {
	w, tk := prEnv(t)
	tk.Doc.Front.SetBlock("github", "  issue: 873\n  repo: acme/widgets")
	if _, err := store.CreateNote(w, "a-secret", "p", model.NoteFinding, "internal failure",
		store.NoteOpts{About: tk.ID, Severity: "major", Body: "/private/operator/token.txt used by a-secret"}); err != nil {
		t.Fatal(err)
	}
	p, err := store.LoadProject(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	p.Doc.Front.Set("github_repo", "acme/widgets")
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	orig := queryRepositoryVisibility
	t.Cleanup(func() { queryRepositoryVisibility = orig })
	queryRepositoryVisibility = func(string, string) (string, error) { return "PUBLIC", nil }

	policy, err := prPublicationPolicy(w, tk, false)
	if err != nil {
		t.Fatal(err)
	}
	body := projectedPRBody(w, tk, policy)
	for _, forbidden := range []string{"internal failure", "/private/operator", "a-secret", "Fixes #873"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public PR leaked %q:\n%s", forbidden, body)
		}
	}
	for _, want := range []string{"Refs #873", "### Acceptance", "Implements dacli task"} {
		if !strings.Contains(body, want) {
			t.Fatalf("public PR omitted %q:\n%s", want, body)
		}
	}
}

func TestPublicPRVerdictsNeedRecordedExactRepoAuthority(t *testing.T) {
	w, tk := prEnv(t)
	p, err := store.LoadProject(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	p.Doc.Front.Set("github_repo", "acme/widgets")
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	orig := queryRepositoryVisibility
	t.Cleanup(func() { queryRepositoryVisibility = orig })
	queryRepositoryVisibility = func(string, string) (string, error) { return "PUBLIC", nil }

	policy, err := prPublicationPolicy(w, tk, true)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Allows(publication.FieldVerdicts) {
		t.Fatalf("unrecorded policy broadened: %+v", policy)
	}
	p, _ = store.LoadProject(w, "p")
	p.Doc.Front.Set("github_internal_disclosure", "another/repo")
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	policy, _ = prPublicationPolicy(w, tk, true)
	if policy.Allows(publication.FieldVerdicts) {
		t.Fatal("authority for a different repo was reused")
	}
	p, _ = store.LoadProject(w, "p")
	p.Doc.Front.Set("github_internal_disclosure", "acme/widgets")
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	policy, _ = prPublicationPolicy(w, tk, true)
	if !policy.Allows(publication.FieldVerdicts) {
		t.Fatalf("exact authority not honored: %+v", policy)
	}
}

func TestUnknownVisibilityUsesPublicSafePRProjection(t *testing.T) {
	w, tk := prEnv(t)
	p, _ := store.LoadProject(w, "p")
	p.Doc.Front.Set("github_repo", "acme/widgets")
	p.Doc.Front.Set("github_internal_disclosure", "acme/widgets")
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	orig := queryRepositoryVisibility
	t.Cleanup(func() { queryRepositoryVisibility = orig })
	queryRepositoryVisibility = func(string, string) (string, error) { return "", fmt.Errorf("offline") }
	policy, err := prPublicationPolicy(w, tk, true)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Visibility != publication.VisibilityUnknown || policy.Allows(publication.FieldFindings) {
		t.Fatalf("unknown visibility did not fail closed: %+v", policy)
	}
}

func TestPublicSafeReuseRemovesExactGeneratedFindings(t *testing.T) {
	w, tk := prEnv(t)
	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "private finding", store.NoteOpts{About: tk.ID, Severity: "major", Body: "/private/operator/evidence"}); err != nil {
		t.Fatal(err)
	}
	current := prBody(w, tk, false) + "\n### Maintainer context\nKeep this.\n"
	policy := publication.New("acme/widgets", "PUBLIC", false, false, false)
	got, action, err := planReusedPRProjection(w, tk, current, false, policy)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "private finding") || strings.Contains(got, "/private/operator") || !strings.Contains(got, "Keep this.") {
		t.Fatalf("reused public projection was not surgically narrowed: action=%q body=%q", action, got)
	}
	if !strings.Contains(action, "generated findings") {
		t.Fatalf("withheld action missing: %q", action)
	}
}

func TestPlanReusedPRBodyRemovesOnlyGeneratedMappedClosingKeyword(t *testing.T) {
	w, tk := prEnv(t)
	tk.Doc.Front.SetBlock("github", "  issue: 841\n  repo: acme/widgets")
	current := "Implements dacli task 001-enrich-pr.\n\nFixes #841\n\nHuman-authored review context must survive.\n"

	got, action, err := planReusedPRBody(w, tk, current, false)
	if err != nil {
		t.Fatal(err)
	}
	if action == "" || strings.Contains(got, "Fixes #841") {
		t.Fatalf("stale closing keyword was not planned for removal: action=%q body=%q", action, got)
	}
	if !strings.Contains(got, "Human-authored review context must survive.") {
		t.Fatalf("reconciliation overwrote unrelated PR content: %q", got)
	}
}

func TestPlanReusedPRBodyRefusesHumanAuthoredClosingKeyword(t *testing.T) {
	w, tk := prEnv(t)
	tk.Doc.Front.SetBlock("github", "  issue: 841\n  repo: acme/widgets")
	_, _, err := planReusedPRBody(w, tk, "Maintainer notes\n\nCloses #841\n", false)
	if err == nil || !strings.Contains(err.Error(), "not recognizably dacli-generated") {
		t.Fatalf("human-authored close directive must fail closed, got %v", err)
	}
}

func TestPlanReusedPRBodyRemovesExactGeneratedVerdictsWithoutHumanContent(t *testing.T) {
	w, tk := prEnv(t)
	tk.Doc.Front.SetBlock("github", "  issue: 841\n  repo: acme/widgets")
	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "race in the merge path",
		store.NoteOpts{About: tk.ID, Severity: "major", Body: "double free at lifecycle.go:200"}); err != nil {
		t.Fatal(err)
	}
	stale := strings.Replace(prBody(w, tk, true), "Implements dacli task 001-enrich-pr.\n", "Implements dacli task 001-enrich-pr.\n\nFixes #841\n", 1) +
		"\n\n### Maintainer context\nKeep this intentional note.\n"

	got, action, err := planReusedPRBody(w, tk, stale, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "Fixes #841") || strings.Contains(got, "TRUST GRADE") {
		t.Fatalf("stale lifecycle/disclosure material survived: action=%q body=%q", action, got)
	}
	if !strings.Contains(got, "### Findings") {
		t.Fatalf("current publication content was removed: %q", got)
	}
	if !strings.Contains(got, "### Maintainer context\nKeep this intentional note.") {
		t.Fatalf("intentional human content was removed: %q", got)
	}
	if !strings.Contains(action, "closing keyword") || !strings.Contains(action, "trust-verdict") {
		t.Fatalf("reconciliation action omitted a change: %q", action)
	}
}

func TestPlanReusedPRBodyRefusesEditedGeneratedVerdictSection(t *testing.T) {
	w, tk := prEnv(t)
	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "race in the merge path",
		store.NoteOpts{About: tk.ID, Severity: "major", Body: "double free at lifecycle.go:200"}); err != nil {
		t.Fatal(err)
	}
	stale := prBody(w, tk, true)
	stale = strings.Replace(stale, "race in the merge path", "maintainer-edited claim", 1)

	_, _, err := planReusedPRBody(w, tk, stale, false)
	if err == nil || !strings.Contains(err.Error(), "cannot be reconciled safely") {
		t.Fatalf("edited sensitive section must fail closed, got %v", err)
	}
}

func TestOpenPRReconcilesStaleGeneratedBodyBeforeReuse(t *testing.T) {
	w, tk := prEnv(t)
	tk.Doc.Front.SetBlock("github", "  issue: 841\n  repo: acme/widgets")
	branch := BranchFor(tk)
	stale := "Implements dacli task 001-enrich-pr.\n\nFixes #841\n\nPreserve this note."
	var edited string
	stubGH(t, func(_ string, args ...string) (string, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "--json url,state"):
			return "OPEN https://github.com/acme/widgets/pull/9", nil
		case strings.Contains(joined, "--json body"):
			return stale, nil
		case strings.HasPrefix(joined, "pr edit "+branch):
			for i := range args {
				if args[i] == "--body" && i+1 < len(args) {
					edited = args[i+1]
				}
			}
			return "", nil
		default:
			return "", nil
		}
	})
	ctx, out := prCtx(w.Root)
	url, reused, err := openPR(ctx, w, "a-root", tk, "main", false, reviewComment, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reused || !strings.Contains(url, "/pull/9") {
		t.Fatalf("reuse = %t url = %q", reused, url)
	}
	if edited == "" || strings.Contains(edited, "Fixes #841") || !strings.Contains(edited, "Preserve this note.") {
		t.Fatalf("edited body = %q", edited)
	}
	if !strings.Contains(out.String(), "reconciled reused PR body") {
		t.Fatalf("operator output omitted reconciliation: %s", out.String())
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
	// ULIDs order chronologically only across millisecond boundaries — the random
	// 80 bits break same-ms ties randomly. A real verify panel votes seconds
	// apart; without this pause both appends can land in one millisecond and the
	// chronological-order assertion below flakes ~50%.
	time.Sleep(2 * time.Millisecond)
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

// TestVerdictReviewLeadsWithTrustGrade covers task 146's other half: the same
// loud trust-grade summary + per-finding tally that --with-verdicts adds to
// the PR body must also lead the posted PR REVIEW, ahead of the existing raw
// per-seat verdict list — enhancing, not replacing, the review's prior content.
func TestVerdictReviewLeadsWithTrustGrade(t *testing.T) {
	w, tk := prEnv(t)

	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "race in the merge path",
		store.NoteOpts{About: tk.ID, Severity: "major", Body: "double free at lifecycle.go:200"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GradeFinding(w, "p", "race in the merge path", "confirmed"); err != nil {
		t.Fatal(err)
	}
	if _, err := eventlog.Append(w, "a-seat1", model.EventComment, tk.ID, "",
		"verify-verdict: confirmed — claude-code (a-seat1) on claim: race in the merge path — reproduces under -race"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := eventlog.Append(w, "a-seat2", model.EventComment, tk.ID, "",
		"verify-verdict: refuted — gemini (a-seat2) on claim: race in the merge path — cannot reproduce"); err != nil {
		t.Fatal(err)
	}

	review := verdictReview(w, tk)
	if !strings.Contains(review, "TRUST GRADE") {
		t.Fatalf("review missing the loud trust-grade section:\n%s", review)
	}
	if !strings.Contains(review, "1 confirmed, 1 refuted") {
		t.Errorf("review missing the per-finding panel vote tally:\n%s", review)
	}
	// Preserved: the existing raw per-seat verdict list must still be present.
	if !strings.Contains(review, "dacli verify panel") || !strings.Contains(review, "confirmed — claude-code") {
		t.Errorf("review lost its existing per-seat verdict list:\n%s", review)
	}
	if i, j := strings.Index(review, "TRUST GRADE"), strings.Index(review, "dacli verify panel"); i < 0 || j < 0 || i > j {
		t.Errorf("trust-grade section must lead the review, ahead of the per-seat list:\n%s", review)
	}
}
