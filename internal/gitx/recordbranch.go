package gitx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/commandresult"
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
	_ = idx.Close()
	// git requires the index file to not exist (or be a valid index); start clean.
	_ = os.Remove(idxPath)
	defer func() { _ = os.Remove(idxPath) }()

	git := func(args ...string) (string, error) {
		ctxArgs := append([]string{"--git-dir", filepath.Join(root, ".git"), "--work-tree", root}, args...)
		c := exec.Command("git", ctxArgs...)
		c.Dir = root
		c.Env = append(os.Environ(), "GIT_INDEX_FILE="+idxPath)
		out, err := commandresult.Run(c, commandresult.RunOptions{Operation: "git " + args[0], WorkspaceRoot: root})
		return strings.TrimSpace(string(out)), err
	}

	// Seed the scratch index from the branch's existing tree when it has one, so
	// `add` produces a delta rather than a first-import every time.
	parent, _ := Run(root, "rev-parse", "-q", "--verify", "refs/heads/"+branch)
	parent = strings.TrimSpace(parent)
	if parent != "" {
		if _, err := git("read-tree", parent); err != nil {
			return "", fmt.Errorf("read-tree %s: %w", branch, err)
		}
	} else {
		if _, err := git("read-tree", "--empty"); err != nil {
			return "", fmt.Errorf("read-tree --empty: %w", err)
		}
	}

	// Stage the path. -A picks up deletions, which matters: a task file moving
	// from open/ to done/ must be a rename in the record, not an orphan copy.
	//
	// When an OUTER .gitignore excludes the whole path — the case dacli 222
	// enables so a generated product repo's trunk never tracks the workspace at
	// all — a plain add would stage nothing (git ignores untracked ignored files,
	// and the first record has an empty parent so every file is untracked), and
	// the record branch would silently stop recording. Force past that outer
	// ignore, but re-apply the path's OWN inner .gitignore (e.g.
	// .dacli/.gitignore's runs/build/worktrees) as exclude pathspecs, so the
	// record stays exactly what it was: everything under the path except its
	// regenerable subtrees. -f alone would sweep those subtrees in, because it
	// overrides the inner ignore too. When the path is NOT outer-ignored the
	// original non-forced add runs unchanged, so an existing `ship
	// --record-branch` records a byte-identical tree.
	addArgs := []string{"add", "-A", "--", path}
	if pathIgnored(root, path) {
		addArgs = []string{"add", "-Af", "--", path}
		for _, sub := range innerIgnoredSubpaths(root, path) {
			addArgs = append(addArgs, ":!"+sub)
		}
	}
	if _, err := git(addArgs...); err != nil {
		return "", fmt.Errorf("staging %s: %w", path, err)
	}
	tree, err := git("write-tree")
	if err != nil {
		return "", fmt.Errorf("write-tree: %w", err)
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
	outB, err := commandresult.Run(c, commandresult.RunOptions{Operation: "git commit-tree", WorkspaceRoot: root})
	commit := strings.TrimSpace(string(outB))
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}
	if _, err := Run(root, "update-ref", "refs/heads/"+branch, commit); err != nil {
		return "", fmt.Errorf("update-ref %s: %w", branch, err)
	}
	return commit, nil
}

// pathIgnored reports whether git's ignore rules exclude path in root's working
// tree — i.e. an OUTER .gitignore lists it. check-ignore exits 0 when a path is
// ignored and 1 when it is not; any other failure (git missing, not a repo) is
// read as "not ignored" so the commit falls back to the original non-forced add
// rather than force-adding on a check that could not run.
func pathIgnored(root, path string) bool {
	c := exec.Command("git", "check-ignore", "-q", "--", path)
	c.Dir = root
	return c.Run() == nil
}

// innerIgnoredSubpaths reads path's own .gitignore (e.g. .dacli/.gitignore) and
// returns the simple directory/file entries it lists, each rejoined under path
// (".dacli/runs", ".dacli/build", ...). These are the subtrees a forced record
// add must still exclude, so routing the workspace to a gitignored trunk does
// not start sweeping transcripts and regenerable build output onto the record
// branch. Only plain, path-anchored entries are honored; a glob or a
// mid-pattern slash is left to the force-add, because a record errs toward
// completeness rather than silently dropping a file a general pattern matched.
func innerIgnoredSubpaths(root, path string) []string {
	raw, err := os.ReadFile(filepath.Join(root, path, ".gitignore"))
	if err != nil {
		return nil
	}
	var subs []string
	for _, line := range strings.Split(string(raw), "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "!") {
			continue
		}
		s = strings.Trim(s, "/")
		if s == "" || strings.ContainsAny(s, `/*?[]\`) {
			continue
		}
		subs = append(subs, path+"/"+s)
	}
	return subs
}
