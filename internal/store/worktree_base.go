package store

import (
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const TaskWorktreeBaseSchema = "task-worktree-base/v1"

// TaskWorktreeBase is the exact observed commit used to create or refresh a
// task worktree. Source distinguishes a fresh remote observation from a local
// fallback in repositories that intentionally have no origin.
type TaskWorktreeBase struct {
	Schema     string `json:"schema"`
	TaskID     string `json:"task_id"`
	Project    string `json:"project"`
	Branch     string `json:"branch"`
	Ref        string `json:"ref"`
	Commit     string `json:"commit"`
	Source     string `json:"source"`
	Configured bool   `json:"configured"`
}

// ResolveTaskWorktreeBase observes the task project's effective landing base.
// If origin exists, observation is fresh and failure is final: silently using
// a stale local ref would contaminate the new branch with unrelated history.
func ResolveTaskWorktreeBase(w *workspace.Workspace, task *Task) (TaskWorktreeBase, error) {
	project, err := LoadProject(w, task.Project)
	if err != nil {
		return TaskWorktreeBase{}, err
	}
	policy, _, err := model.ResolveLanding(project.Landing, model.LandingOverride{})
	if err != nil {
		return TaskWorktreeBase{}, err
	}
	branch := strings.TrimSpace(policy.Base)
	configured := branch != ""
	if branch == "" {
		branch = TrunkBranch(w)
	}
	if branch == "" {
		return TaskWorktreeBase{}, fmt.Errorf("cannot resolve a landing base for task %s", task.ID)
	}
	base := TaskWorktreeBase{Schema: TaskWorktreeBaseSchema, TaskID: task.ID, Project: task.Project, Branch: branch, Configured: configured}

	remotes, err := gitx.Run(w.Root, "remote")
	if err != nil {
		return TaskWorktreeBase{}, fmt.Errorf("list git remotes for worktree base: %w", err)
	}
	for _, remote := range strings.Fields(remotes) {
		if remote != "origin" {
			continue
		}
		if _, err := gitx.Run(w.Root, "fetch", "--no-tags", "origin", "refs/heads/"+branch+":refs/remotes/origin/"+branch); err != nil {
			return TaskWorktreeBase{}, fmt.Errorf("observe landing base origin/%s: %w", branch, err)
		}
		base.Ref = "refs/remotes/origin/" + branch
		base.Source = "fresh-origin"
		base.Commit, err = gitx.Run(w.Root, "rev-parse", "--verify", base.Ref+"^{commit}")
		if err != nil {
			return TaskWorktreeBase{}, fmt.Errorf("resolve observed landing base origin/%s: %w", branch, err)
		}
		base.Commit = strings.TrimSpace(base.Commit)
		return base, nil
	}

	base.Ref = "refs/heads/" + branch
	base.Source = "local-no-origin"
	base.Commit, err = gitx.Run(w.Root, "rev-parse", "--verify", base.Ref+"^{commit}")
	if err != nil {
		return TaskWorktreeBase{}, fmt.Errorf("resolve local landing base %s: %w", branch, err)
	}
	base.Commit = strings.TrimSpace(base.Commit)
	return base, nil
}
