package execution

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const reviewTranscriptTail = 256 << 10

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
		return err
	}
	for _, event := range events {
		var recorded store.IndependentReviewResult
		if json.Unmarshal([]byte(event.Body), &recorded) == nil && validateObservedReview(w, t, childID, role, runtime, modelName, recorded) == nil {
			return nil
		}
	}
	transcript, err := readReviewTranscriptTail(transcriptPath)
	if err != nil {
		return fmt.Errorf("read review transcript: %w", err)
	}
	result, err := store.ParseReviewOutput(transcript)
	if err != nil {
		return err
	}
	if err := validateObservedReview(w, t, childID, role, runtime, modelName, result); err != nil {
		return err
	}
	raw, _ := json.Marshal(result)
	_, err = eventlog.Append(w, childID, model.EventReview, t.ID, store.ReviewResultSchema, string(raw))
	return err
}
