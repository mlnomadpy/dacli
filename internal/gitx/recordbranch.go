package gitx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommitPathToBranch commits ONE path (e.g. ".dacli") onto its own branch,
// without touching the working tree, the index, or the checked-out branch.
//
// It exists because of what committing the workspace record onto trunk does to
// a repository's history. In dacli's own repo, 251 of 429 commits were
// bookkeeping, one message appeared verbatim 61 times ("record workspace after
// integrating 0 task(s)"), and 30% of recent commits touched nothing but
// `.dacli/`. A reader — human or model — looking for engineering history finds
// mostly a loop narrating itself. Routing the record to its own ref keeps the
// full trajectory in the repository while leaving trunk's history to be what it
// claims: the code, and why it changed (dacli 193).
//
// Implementation is git plumbing over a TEMPORARY index file, so no checkout,
// stash, or worktree is involved and a concurrent agent's staged work is never
// disturbed: read-tree into a scratch index, add the path, write-tree,
// commit-tree onto the branch's current tip, update-ref.
//
// Returns the new commit sha, or "" when the path is unchanged since the last
// record — an unchanged workspace produces no commit rather than an empty one.
func CommitPathToBranch(root, branch, path, message, authorName, authorEmail string) (string, error) {
	idx, err := os.CreateTemp("", "dacli-record-index-*")
	if err != nil {
		return "", fmt.Errorf("record index: %w", err)
	}
	idxPath := idx.Name()
	idx.Close()
	// git requires the index file to not exist (or be a valid index); start clean.
	os.Remove(idxPath)
	defer os.Remove(idxPath)

	git := func(args ...string) (string, error) {
		ctxArgs := append([]string{"--git-dir", filepath.Join(root, ".git"), "--work-tree", root}, args...)
		c := exec.Command("git", ctxArgs...)
		c.Dir = root
		c.Env = append(os.Environ(), "GIT_INDEX_FILE="+idxPath)
		out, err := c.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	// Seed the scratch index from the branch's existing tree when it has one, so
	// `add` produces a delta rather than a first-import every time.
	parent, _ := Run(root, "rev-parse", "-q", "--verify", "refs/heads/"+branch)
	parent = strings.TrimSpace(parent)
	if parent != "" {
		if out, err := git("read-tree", parent); err != nil {
			return "", fmt.Errorf("read-tree %s: %s", branch, out)
		}
	} else {
		if out, err := git("read-tree", "--empty"); err != nil {
			return "", fmt.Errorf("read-tree --empty: %s", out)
		}
	}

	// Stage the path. -A picks up deletions, which matters: a task file moving
	// from open/ to done/ must be a rename in the record, not an orphan copy.
	if out, err := git("add", "-A", "--", path); err != nil {
		return "", fmt.Errorf("staging %s: %s", path, out)
	}
	tree, err := git("write-tree")
	if err != nil {
		return "", fmt.Errorf("write-tree: %s", tree)
	}

	// Unchanged since the last record: say nothing rather than committing noise.
	if parent != "" {
		if prevTree, terr := Run(root, "rev-parse", parent+"^{tree}"); terr == nil {
			if strings.TrimSpace(prevTree) == tree {
				return "", nil
			}
		}
	}

	args := []string{
		"-c", "user.name=" + authorName, "-c", "user.email=" + authorEmail,
		"commit-tree", tree, "-m", message,
	}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	c := exec.Command("git", args...)
	c.Dir = root
	stamp := time.Now().Format(time.RFC3339)
	c.Env = append(os.Environ(),
		"GIT_INDEX_FILE="+idxPath,
		"GIT_AUTHOR_NAME="+authorName, "GIT_AUTHOR_EMAIL="+authorEmail, "GIT_AUTHOR_DATE="+stamp,
		"GIT_COMMITTER_NAME="+authorName, "GIT_COMMITTER_EMAIL="+authorEmail, "GIT_COMMITTER_DATE="+stamp,
	)
	outB, err := c.CombinedOutput()
	commit := strings.TrimSpace(string(outB))
	if err != nil {
		return "", fmt.Errorf("commit-tree: %s", commit)
	}
	if out, err := Run(root, "update-ref", "refs/heads/"+branch, commit); err != nil {
		return "", fmt.Errorf("update-ref %s: %s", branch, out)
	}
	return commit, nil
}
