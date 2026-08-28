package orchestration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/verifyroute"
)

// verifyTaskChange executes the repository-declared gates against the task's
// committed isolated checkout. Running them from the controller root would
// verify trunk instead of the worker's branch, while shell strings would make
// paths with spaces executable syntax (issue #860).
func (d *driver) verifyTaskChange(task *store.Task) error {
	profile, err := loadProfile(d.w, d.cfg.project)
	if errors.Is(err, os.ErrNotExist) {
		return nil // direct-loop and legacy-profile compatibility
	}
	if err != nil {
		return err
	}
	if len(profile.Verification.Rules) == 0 {
		return nil // legacy-profile compatibility
	}
	branch := taskBranch(task)
	worktree, err := worktreeForBranch(d.w.Root, branch)
	if err != nil {
		return err
	}
	base := d.trunkBase()
	if base == "" {
		base = d.trunkBranch
	}
	out, err := gitx.Run(d.w.Root, "diff", "--name-only", base+"..."+branch)
	if err != nil {
		return fmt.Errorf("resolve changed paths for %s against %s: %w", branch, base, err)
	}
	var paths []string
	for _, name := range strings.Split(out, "\n") {
		if name = strings.TrimSpace(name); name != "" {
			paths = append(paths, name)
		}
	}
	commands, err := verifyroute.Resolve(worktree, profile.Verification.Rules, profile.Verification.ContractGroups, paths)
	if err != nil {
		return fmt.Errorf("route verification for %s: %w", task.ID, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.workerTimeout(task))*time.Second)
	defer cancel()
	results, err := verifyroute.Execute(ctx, worktree, commands)
	for _, result := range results {
		d.logf("    %03d: verified rule %s · cwd=%s · argv=%q", task.Seq, result.RuleID, result.Cwd, result.Argv)
	}
	if err != nil {
		return err
	}
	if len(commands) == 0 {
		d.logf("    %03d: changed paths require no configured gate", task.Seq)
	}
	return nil
}

func worktreeForBranch(root, branch string) (string, error) {
	worktrees, err := gitx.ListWorktrees(root)
	if err != nil {
		return "", fmt.Errorf("list worktrees for verification: %w", err)
	}
	for _, worktree := range worktrees {
		if worktree.Branch == branch {
			return worktree.Path, nil
		}
	}
	return "", fmt.Errorf("verification requires the isolated checkout for branch %s; the worktree is missing", branch)
}
