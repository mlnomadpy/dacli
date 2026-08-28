package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

const ReleaseTrainSchema = "release-train/v1"

// ReleaseTrain is the durable transaction for one exact source-to-target
// promotion. It is deliberately branch-pair keyed: a restart must resume the
// PR whose identity was observed before it considers creating another one.
type ReleaseTrain struct {
	Schema          string    `json:"schema"`
	Project         string    `json:"project"`
	Source          string    `json:"source"`
	Target          string    `json:"target"`
	SourceSHA       string    `json:"source_sha"`
	TargetSHA       string    `json:"target_sha"`
	RequiredChecks  []string  `json:"required_checks,omitempty"`
	RequiredReviews int       `json:"required_reviews,omitempty"`
	IncludedTasks   []string  `json:"included_tasks,omitempty"`
	ReconciledTasks []string  `json:"reconciled_tasks,omitempty"`
	PullRequest     int       `json:"pull_request,omitempty"`
	PullRequestURL  string    `json:"pull_request_url,omitempty"`
	Phase           string    `json:"phase"`
	LandedTargetSHA string    `json:"landed_target_sha,omitempty"`
	CleanupComplete bool      `json:"cleanup_complete,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func releaseTrainPath(w *workspace.Workspace, project, source, target string) (string, error) {
	if !workspace.SafeSegment(project) {
		return "", fmt.Errorf("unsafe release-train project %q", project)
	}
	if strings.TrimSpace(source) == "" || strings.TrimSpace(target) == "" {
		return "", fmt.Errorf("release-train branches are required")
	}
	// Branches may legitimately contain slashes. Hash the exact pair instead of
	// turning branch syntax into filesystem syntax (and keep the originals in
	// the validated transaction so collisions cannot be adopted).
	key := sha256.Sum256([]byte(source + "\x00" + target))
	return filepath.Join(w.Root, workspace.Dir, "release-trains", project, fmt.Sprintf("%x.json", key[:12])), nil
}

func ReadReleaseTrain(w *workspace.Workspace, project, source, target string) (ReleaseTrain, error) {
	path, err := releaseTrainPath(w, project, source, target)
	if err != nil {
		return ReleaseTrain{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ReleaseTrain{}, err
	}
	var tx ReleaseTrain
	if err := json.Unmarshal(b, &tx); err != nil {
		return tx, fmt.Errorf("decode release train: %w", err)
	}
	if tx.Schema != ReleaseTrainSchema || tx.Project != project || tx.Source != source || tx.Target != target || tx.Phase == "" {
		return tx, fmt.Errorf("invalid release train for %s %s->%s", project, source, target)
	}
	return tx, nil
}

func WriteReleaseTrain(w *workspace.Workspace, tx ReleaseTrain) error {
	path, err := releaseTrainPath(w, tx.Project, tx.Source, tx.Target)
	if err != nil {
		return err
	}
	if tx.Schema != ReleaseTrainSchema || tx.Phase == "" || strings.TrimSpace(tx.SourceSHA) == "" || strings.TrimSpace(tx.TargetSHA) == "" {
		return fmt.Errorf("invalid release train transaction")
	}
	tx.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".release-train-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
