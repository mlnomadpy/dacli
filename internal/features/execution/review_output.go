package execution

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const reviewTranscriptTail = 256 << 10

var appendReviewResultEvent = eventlog.Append

func readReviewTranscriptTail(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > reviewTranscriptTail {
		if _, err := f.Seek(-reviewTranscriptTail, io.SeekEnd); err != nil {
			return "", err
		}
	}
	raw, err := io.ReadAll(f)
	return string(raw), err
}

func validateObservedReview(w *workspace.Workspace, t *store.Task, childID string, role team.Role, runtime, modelName string, result store.IndependentReviewResult) error {
	if err := result.Validate(); err != nil {
		return err
	}
	if result.ReviewerID != childID || result.ReviewerRole != role.Name || result.Runtime != runtime || result.Model != modelName || result.Grant != "ro" {
		return fmt.Errorf("review envelope identity/runtime/model/grant does not match the launched reviewer")
	}
	branch := fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug)
	commit, err := gitx.Run(w.Root, "rev-parse", "--verify", branch)
	if err != nil {
		return fmt.Errorf("observe reviewed branch %s: %w", branch, err)
	}
	commit = strings.TrimSpace(commit)
	tree, err := gitx.Run(w.Root, "rev-parse", "--verify", commit+"^{tree}")
	if err != nil {
		return fmt.Errorf("observe reviewed tree %s: %w", branch, err)
	}
	if result.CommitSHA != commit || result.TreeSHA != strings.TrimSpace(tree) {
		return fmt.Errorf("stale review envelope: reviewed %s/%s, current %s/%s", result.CommitSHA, result.TreeSHA, commit, strings.TrimSpace(tree))
	}
	return nil
}

// materializeReviewOutput converts provider output into the append-only event
// from the parent process. A Codex `--sandbox read-only` reviewer therefore
// reports without being granted a filesystem exception just to write .dacli.
func materializeReviewOutput(w *workspace.Workspace, t *store.Task, childID string, role team.Role, runtime, modelName, transcriptPath string) error {
	// The explicit `review record` command is the equivalent path for runtimes
	// whose RO contract permits append-only dacli calls. Accept it only after
	// revalidating the same exact identity and tree contract.
	events, err := eventlog.List(w, eventlog.Query{Actor: childID, About: t.ID, Kinds: []model.EventKind{model.EventReview}})
	if err != nil {
		return requestReviewResultHandoff(w, transcriptPath, "review_result_observation_failure", store.IndependentReviewResult{}, err)
	}
	for _, event := range events {
		var recorded store.IndependentReviewResult
		if json.Unmarshal([]byte(event.Body), &recorded) == nil && validateObservedReview(w, t, childID, role, runtime, modelName, recorded) == nil {
			return nil
		}
	}
	transcript, err := readReviewTranscriptTail(transcriptPath)
	if err != nil {
		cause := fmt.Errorf("read review transcript: %w", err)
		return requestReviewResultHandoff(w, transcriptPath, "review_result_observation_failure", store.IndependentReviewResult{}, cause)
	}
	result, err := store.ParseReviewOutput(transcript)
	if err != nil {
		return requestReviewResultHandoff(w, transcriptPath, "review_result_protocol_failure", store.IndependentReviewResult{}, err)
	}
	if err := validateObservedReview(w, t, childID, role, runtime, modelName, result); err != nil {
		return requestReviewResultHandoff(w, transcriptPath, "review_result_validation_failure", store.IndependentReviewResult{}, err)
	}
	raw, _ := json.Marshal(result)
	if _, err = appendReviewResultEvent(w, childID, model.EventReview, t.ID, store.ReviewResultSchema, string(raw)); err != nil {
		return requestReviewResultHandoff(w, transcriptPath, "review_result_publication_failure", result, err)
	}
	return nil
}

func requestReviewResultHandoff(w *workspace.Workspace, transcriptPath, failureClass string, result store.IndependentReviewResult, cause error) error {
	runID := filepath.Base(filepath.Dir(transcriptPath))
	unresolved := []string{"structured independent-review verdict is unavailable; no approval may be inferred"}
	if len(result.Findings) > 0 {
		unresolved = unresolved[:0]
		for _, finding := range result.Findings {
			label := strings.TrimSpace(finding.ID)
			if label == "" {
				label = "unidentified-review-finding"
			}
			unresolved = append(unresolved, fmt.Sprintf("%s %s %s:%d — %s", label, finding.Severity, finding.File, finding.Line, finding.AffectedInvariant))
		}
	}
	nextAction := "owner corrects the governed review-result failure and re-runs independent review on the exact current tree; do not inspect raw transcript or treat silence as approval"
	if failureClass == "review_result_publication_failure" {
		nextAction = "owner restores the governed review-result channel and re-runs independent review on the exact current tree; do not inspect raw transcript or treat silence as approval"
	}
	req := store.RootHandoffRequest{
		Schema: store.RootHandoffSchema, Unresolved: unresolved,
		FailedOperation: "persist structured independent-review result", FailureClass: failureClass, Stderr: cause.Error(),
		NextAction: nextAction,
	}
	if err := store.WriteRootHandoffRequest(w, runID, req); err != nil {
		return fmt.Errorf("%w; write review-result handoff: %w", cause, err)
	}
	return fmt.Errorf("%w; handoff-required for run %s", cause, runID)
}
