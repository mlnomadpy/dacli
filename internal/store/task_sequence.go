package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const taskSequenceStateSchema = "task-sequence/v1"

type taskSequenceState struct {
	Schema      string `json:"schema"`
	Next        int    `json:"next"`
	Observation string `json:"observation"`
}

func taskSequenceStatePath(w *workspace.Workspace, project string) string {
	return filepath.Join(w.ProjectDir(project), ".task-sequence.json")
}

// nextTaskSequence is called under the project's cross-process seq lock. The
// small state file is only an acceleration index: directory/tombstone/Git-ref
// observations must match before it is trusted, and any malformed or stale
// state rebuilds from the canonical filenames, tombstones, and Git history.
func nextTaskSequence(w *workspace.Workspace, project string) (int, error) {
	observation := observeTaskSequenceInputs(w, project)
	if raw, err := os.ReadFile(taskSequenceStatePath(w, project)); err == nil {
		var state taskSequenceState
		if json.Unmarshal(raw, &state) == nil && state.Schema == taskSequenceStateSchema && state.Next > 0 && state.Observation == observation {
			return state.Next, nil
		}
	}

	for attempt := 0; attempt < 3; attempt++ {
		before := observeTaskSequenceInputs(w, project)
		ceiling := onDiskSeqCeiling(w, project)
		if gitCeiling := gitTaskSeqCeiling(w, project); gitCeiling > ceiling {
			ceiling = gitCeiling
		}
		if removed := TombstoneSeqCeiling(w, project); removed > ceiling {
			ceiling = removed
		}
		after := observeTaskSequenceInputs(w, project)
		if before == after {
			return ceiling + 1, nil
		}
	}
	return 0, fmt.Errorf("task sequence inputs changed repeatedly while allocating for project %s; retry after concurrent filesystem/Git activity settles", project)
}

func recordNextTaskSequence(w *workspace.Workspace, project string, next int) {
	state := taskSequenceState{Schema: taskSequenceStateSchema, Next: next, Observation: observeTaskSequenceInputs(w, project)}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	raw = append(raw, '\n')
	// Acceleration metadata is recoverable. If this write fails, the task file
	// remains canonical and the next allocation safely rebuilds the state.
	_ = mdstore.WriteBytes(taskSequenceStatePath(w, project), raw, 0o644)
}

func observeTaskSequenceInputs(w *workspace.Workspace, project string) string {
	parts := make([]string, 0, len(model.AllStatuses)+3)
	for _, status := range model.AllStatuses {
		parts = append(parts, sequencePathObservation(w.TasksDir(project, status)))
	}
	parts = append(parts, sequencePathObservation(w.TombstonesDir(project)))
	parts = append(parts, observeGitRefs(w.Root))
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return fmt.Sprintf("%x", sum)
}

// observeGitRefs reads the small ref files directly instead of starting a Git
// process for every task creation. Ref contents, packed-refs, and linked
// worktree common-dir resolution are enough to invalidate the acceleration
// state whenever the history ceiling may have changed; the expensive git-log
// ceiling is then recomputed once on that invalidation.
func observeGitRefs(root string) string {
	marker := filepath.Join(root, ".git")
	info, err := os.Stat(marker)
	if err != nil {
		return "git:not-a-repository"
	}
	gitDir := marker
	if !info.IsDir() {
		raw, readErr := os.ReadFile(marker)
		if readErr != nil || !strings.HasPrefix(strings.TrimSpace(string(raw)), "gitdir:") {
			return "git:unreadable-marker"
		}
		gitDir = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(raw)), "gitdir:"))
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(root, gitDir)
		}
	}
	commonDir := gitDir
	if raw, readErr := os.ReadFile(filepath.Join(gitDir, "commondir")); readErr == nil {
		commonDir = strings.TrimSpace(string(raw))
		if !filepath.IsAbs(commonDir) {
			commonDir = filepath.Join(gitDir, commonDir)
		}
	}
	var records []string
	for _, single := range []string{filepath.Join(commonDir, "packed-refs"), filepath.Join(gitDir, "HEAD")} {
		if raw, readErr := os.ReadFile(single); readErr == nil {
			records = append(records, single+":"+string(raw))
		}
	}
	var refPaths []string
	_ = filepath.WalkDir(filepath.Join(commonDir, "refs"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		refPaths = append(refPaths, path)
		return nil
	})
	for _, refPath := range refPaths {
		if raw, readErr := os.ReadFile(refPath); readErr == nil { // #nosec G304 -- paths are enumerated beneath Git's resolved refs directory
			records = append(records, refPath+":"+string(raw))
		}
	}
	sort.Strings(records)
	return "git-refs:" + strings.Join(records, "\n")
}

func sequencePathObservation(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return path + ":" + err.Error()
	}
	return fmt.Sprintf("%s:%d:%d:%d", path, info.ModTime().UTC().UnixNano(), info.Size(), info.Mode())
}
