package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

const (
	ReviewResultSchema     = "independent-review-result/v1"
	ReviewJournalSchema    = "independent-review-transaction/v1"
	ReviewOutputMarker     = "DACLI_REVIEW_RESULT "
	ReviewValidationSchema = "independent-review-validation/v1"
)

type ReviewVerdict string

const (
	ReviewApprove               ReviewVerdict = "approve"
	ReviewRequestChanges        ReviewVerdict = "request-changes"
	ReviewInconclusive          ReviewVerdict = "inconclusive"
	ReviewNoResponse            ReviewVerdict = "no-response"
	ReviewInfrastructureFailure ReviewVerdict = "infrastructure-failure"
)

type ReviewFinding struct {
	ID                    string `json:"id"`
	Severity              string `json:"severity"`
	File                  string `json:"file"`
	Line                  int    `json:"line,omitempty"`
	EndLine               int    `json:"end_line,omitempty"`
	Evidence              string `json:"evidence"`
	AffectedInvariant     string `json:"affected_invariant"`
	SuggestedVerification string `json:"suggested_verification"`
}

type IndependentReviewResult struct {
	Schema               string          `json:"schema"`
	Verdict              ReviewVerdict   `json:"verdict"`
	Findings             []ReviewFinding `json:"findings"`
	ReviewerID           string          `json:"reviewer_id"`
	ReviewerRole         string          `json:"reviewer_role"`
	Runtime              string          `json:"runtime"`
	Model                string          `json:"model,omitempty"`
	Grant                string          `json:"grant"`
	IndependentOf        []string        `json:"independent_of"`
	CommitSHA            string          `json:"commit_sha"`
	TreeSHA              string          `json:"tree_sha"`
	ObservedPRGeneration int             `json:"observed_pr_generation,omitempty"`
	ObservedAt           time.Time       `json:"observed_at"`
	Detail               string          `json:"detail,omitempty"`
}

type ReviewValidationIdentity struct {
	Schema       string `json:"schema"`
	ReviewerID   string `json:"reviewer_id"`
	ReviewerRole string `json:"reviewer_role"`
	Runtime      string `json:"runtime"`
	Model        string `json:"model"`
	Grant        string `json:"grant"`
	CommitSHA    string `json:"commit_sha"`
	TreeSHA      string `json:"tree_sha"`
}

type ReviewValidationDiagnostic struct {
	Schema     string                   `json:"schema"`
	Mismatches []string                 `json:"mismatches"`
	Expected   ReviewValidationIdentity `json:"expected"`
	Actual     ReviewValidationIdentity `json:"actual"`
}

type ReviewValidationError struct {
	Diagnostic ReviewValidationDiagnostic
	cause      error
}

func (e *ReviewValidationError) Error() string {
	return fmt.Sprintf("independent review validation mismatch (%s)", strings.Join(e.Diagnostic.Mismatches, ", "))
}

func (e *ReviewValidationError) Unwrap() error { return e.cause }

func ReviewValidationIdentityOf(result IndependentReviewResult) ReviewValidationIdentity {
	return ReviewValidationIdentity{
		Schema: result.Schema, ReviewerID: result.ReviewerID, ReviewerRole: result.ReviewerRole,
		Runtime: result.Runtime, Model: result.Model, Grant: result.Grant,
		CommitSHA: result.CommitSHA, TreeSHA: result.TreeSHA,
	}
}

func NewReviewValidationError(expected, actual IndependentReviewResult, mismatches []string, cause error) error {
	return &ReviewValidationError{Diagnostic: ReviewValidationDiagnostic{
		Schema: ReviewValidationSchema, Mismatches: append([]string(nil), mismatches...),
		Expected: ReviewValidationIdentityOf(expected), Actual: ReviewValidationIdentityOf(actual),
	}, cause: cause}
}

var stableFindingID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{2,127}$`)

func (r IndependentReviewResult) Validate() error {
	if r.Schema != ReviewResultSchema {
		return fmt.Errorf("unsupported review result schema %q", r.Schema)
	}
	if !slices.Contains([]ReviewVerdict{ReviewApprove, ReviewRequestChanges, ReviewInconclusive, ReviewNoResponse, ReviewInfrastructureFailure}, r.Verdict) {
		return fmt.Errorf("unknown review verdict %q", r.Verdict)
	}
	if r.ReviewerID == "" || r.ReviewerRole == "" || r.Runtime == "" {
		return fmt.Errorf("reviewer identity, role, and runtime are required")
	}
	if r.Grant != "ro" {
		return fmt.Errorf("independent reviewer grant must be ro, got %q", r.Grant)
	}
	if len(r.IndependentOf) == 0 {
		return fmt.Errorf("independence relationship is required")
	}
	if slices.Contains(r.IndependentOf, r.ReviewerID) {
		return fmt.Errorf("reviewer cannot be independent of itself")
	}
	if r.CommitSHA == "" || r.TreeSHA == "" {
		return fmt.Errorf("review result must bind the exact commit and tree")
	}
	if r.ObservedAt.IsZero() {
		return fmt.Errorf("review observation timestamp is required")
	}
	seen := map[string]bool{}
	for i, finding := range r.Findings {
		if !stableFindingID.MatchString(finding.ID) || seen[finding.ID] {
			return fmt.Errorf("finding %d has missing, unstable, or duplicate id %q", i, finding.ID)
		}
		seen[finding.ID] = true
		if !slices.Contains([]string{"major", "moderate", "minor"}, finding.Severity) {
			return fmt.Errorf("finding %s has invalid severity %q", finding.ID, finding.Severity)
		}
		cleanFile := filepath.Clean(finding.File)
		if finding.File == "" || filepath.IsAbs(finding.File) || strings.Contains(finding.File, "\\") || cleanFile != finding.File || cleanFile == ".." || strings.HasPrefix(cleanFile, ".."+string(filepath.Separator)) {
			return fmt.Errorf("finding %s has unsafe or missing repository-relative file", finding.ID)
		}
		if finding.Line < 0 || finding.EndLine < 0 || finding.EndLine > 0 && finding.EndLine < finding.Line {
			return fmt.Errorf("finding %s has invalid line range", finding.ID)
		}
		if finding.Evidence == "" || finding.AffectedInvariant == "" || finding.SuggestedVerification == "" {
			return fmt.Errorf("finding %s needs evidence, affected invariant, and suggested verification", finding.ID)
		}
	}
	if r.Verdict == ReviewRequestChanges && len(r.Findings) == 0 {
		return fmt.Errorf("request-changes requires at least one structured finding")
	}
	if r.Verdict == ReviewApprove && len(r.Findings) > 0 {
		return fmt.Errorf("approve cannot carry unresolved findings")
	}
	return nil
}

func (r IndependentReviewResult) IsApproval() bool {
	return r.Validate() == nil && r.Verdict == ReviewApprove
}

// ParseReviewOutput extracts the terminal one-line review envelope. It is the
// return path for genuinely read-only sandboxes: the parent materializes the
// event after the provider exits, so the reviewer never needs filesystem write
// authority merely to report its verdict.
func ParseReviewOutput(transcript string) (IndependentReviewResult, error) {
	result, err := DecodeReviewOutput(transcript)
	if err != nil {
		return result, err
	}
	if err := result.Validate(); err != nil {
		return result, err
	}
	return result, nil
}

// DecodeReviewOutput parses the envelope without validating its claims. The
// parent launch path uses this so schema/grant/identity mismatches can retain
// the complete expected/actual diagnostic instead of collapsing early.
func DecodeReviewOutput(transcript string) (IndependentReviewResult, error) {
	var result IndependentReviewResult
	index := strings.LastIndex(transcript, ReviewOutputMarker)
	if index < 0 {
		return result, fmt.Errorf("missing %s envelope", strings.TrimSpace(ReviewOutputMarker))
	}
	raw := transcript[index+len(ReviewOutputMarker):]
	if end := strings.IndexByte(raw, '\n'); end >= 0 {
		raw = raw[:end]
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &result); err != nil {
		return result, fmt.Errorf("decode %s envelope: %w", strings.TrimSpace(ReviewOutputMarker), err)
	}
	return result, nil
}

// PublicReviewProjection is derived from the same typed result used by the
// loop, but deliberately omits evidence, local paths outside repository-relative
// locations, identities, runtime/model data, and independence metadata.
type PublicReviewProjection struct {
	Verdict  ReviewVerdict         `json:"verdict"`
	Commit   string                `json:"commit_sha"`
	Tree     string                `json:"tree_sha"`
	Comments []PublicReviewComment `json:"comments,omitempty"`
}

type PublicReviewComment struct {
	FindingID string `json:"finding_id"`
	Path      string `json:"path"`
	Line      int    `json:"line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
	Body      string `json:"body"`
}

func (r IndependentReviewResult) PublicProjection() (PublicReviewProjection, error) {
	if err := r.Validate(); err != nil {
		return PublicReviewProjection{}, err
	}
	p := PublicReviewProjection{Verdict: r.Verdict, Commit: r.CommitSHA, Tree: r.TreeSHA}
	for _, finding := range r.Findings {
		p.Comments = append(p.Comments, PublicReviewComment{FindingID: finding.ID, Path: finding.File, Line: finding.Line, EndLine: finding.EndLine,
			Body: fmt.Sprintf("[%s] %s\n\nSuggested verification: %s", finding.ID, finding.AffectedInvariant, finding.SuggestedVerification)})
	}
	return p, nil
}

type ReviewTransactionState string

const (
	ReviewAwaiting         ReviewTransactionState = "awaiting-review"
	ReviewCorrection       ReviewTransactionState = "correction-required"
	ReviewAwaitingRereview ReviewTransactionState = "awaiting-re-review"
	ReviewApproved         ReviewTransactionState = "approved"
	ReviewHalted           ReviewTransactionState = "halted"
)

type ReviewTransaction struct {
	Schema          string                 `json:"schema"`
	Project         string                 `json:"project"`
	TaskID          string                 `json:"task_id"`
	Branch          string                 `json:"branch"`
	State           ReviewTransactionState `json:"state"`
	CorrectionTurns int                    `json:"correction_turns"`
	MaxCorrections  int                    `json:"max_corrections"`
	PriorCommit     string                 `json:"prior_commit"`
	PriorTree       string                 `json:"prior_tree"`
	CurrentCommit   string                 `json:"current_commit"`
	CurrentTree     string                 `json:"current_tree"`
	FindingIDs      []string               `json:"finding_ids,omitempty"`
	ReviewRunID     string                 `json:"review_run_id,omitempty"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

func (t *ReviewTransaction) Apply(result IndependentReviewResult, observedCommit, observedTree string, now time.Time) error {
	if err := result.Validate(); err != nil {
		t.State = ReviewHalted
		return err
	}
	if observedCommit == "" || observedTree == "" || result.CommitSHA != observedCommit || result.TreeSHA != observedTree {
		t.State = ReviewHalted
		return fmt.Errorf("stale review result: reviewed commit/tree %s/%s, current %s/%s", result.CommitSHA, result.TreeSHA, observedCommit, observedTree)
	}
	if t.State == ReviewAwaitingRereview && result.TreeSHA == t.PriorTree {
		t.State = ReviewHalted
		return fmt.Errorf("re-review observed the prior tree %s instead of corrected tree", result.TreeSHA)
	}
	t.CurrentCommit, t.CurrentTree, t.UpdatedAt = observedCommit, observedTree, now.UTC()
	switch result.Verdict {
	case ReviewApprove:
		t.State, t.FindingIDs = ReviewApproved, nil
	case ReviewRequestChanges:
		if t.CorrectionTurns >= t.MaxCorrections {
			t.State = ReviewHalted
			return fmt.Errorf("review correction limit reached (%d/%d)", t.CorrectionTurns, t.MaxCorrections)
		}
		t.CorrectionTurns++
		t.PriorCommit, t.PriorTree = observedCommit, observedTree
		t.FindingIDs = t.FindingIDs[:0]
		for _, finding := range result.Findings {
			t.FindingIDs = append(t.FindingIDs, finding.ID)
		}
		slices.Sort(t.FindingIDs)
		t.State = ReviewCorrection
	case ReviewInconclusive, ReviewNoResponse, ReviewInfrastructureFailure:
		t.State = ReviewHalted
		return fmt.Errorf("review did not approve: %s", result.Verdict)
	}
	return nil
}

func (t *ReviewTransaction) MarkCorrected(commit, tree string, now time.Time) error {
	if t.State != ReviewCorrection {
		return fmt.Errorf("cannot record correction from review state %s", t.State)
	}
	if commit == "" || tree == "" || tree == t.PriorTree {
		return fmt.Errorf("correction must produce a new exact commit/tree")
	}
	t.CurrentCommit, t.CurrentTree, t.State, t.UpdatedAt = commit, tree, ReviewAwaitingRereview, now.UTC()
	return nil
}

func ReviewTransactionPath(w *workspace.Workspace, project, taskID string) string {
	return filepath.Join(w.Root, workspace.Dir, "loop", "reviews", project, taskID+".json")
}

func WriteReviewTransaction(w *workspace.Workspace, tx ReviewTransaction) error {
	if !workspace.SafeSegment(tx.Project) || !workspace.SafeSegment(tx.TaskID) {
		return fmt.Errorf("unsafe review transaction identity %q/%q", tx.Project, tx.TaskID)
	}
	tx.Schema = ReviewJournalSchema
	raw, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	path := ReviewTransactionPath(w, tx.Project, tx.TaskID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return atomicWrite(path, append(raw, '\n'))
}

func ReadReviewTransaction(w *workspace.Workspace, project, taskID string) (ReviewTransaction, error) {
	var tx ReviewTransaction
	if !workspace.SafeSegment(project) || !workspace.SafeSegment(taskID) {
		return tx, fmt.Errorf("unsafe review transaction identity %q/%q", project, taskID)
	}
	raw, err := os.ReadFile(ReviewTransactionPath(w, project, taskID))
	if err != nil {
		return tx, err
	}
	if err := json.Unmarshal(raw, &tx); err != nil {
		return tx, err
	}
	validState := slices.Contains([]ReviewTransactionState{ReviewAwaiting, ReviewCorrection, ReviewAwaitingRereview, ReviewApproved, ReviewHalted}, tx.State)
	if tx.Schema != ReviewJournalSchema || tx.Project != project || tx.TaskID != taskID || tx.Branch == "" || tx.MaxCorrections < 0 || !validState {
		return tx, fmt.Errorf("invalid review transaction for %s", taskID)
	}
	return tx, nil
}

func atomicWrite(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".review-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
