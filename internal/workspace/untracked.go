package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultRecordBranch is the ref an untracked workspace records its
// trajectory to. One well-known name keeps `git log dacli-record`
// discoverable without configuration; an operator who wants another passes
// `ship --record-branch`.
const DefaultRecordBranch = "dacli-record"

// UntrackFromTrunk excludes the workspace from the product repo's trunk AND
// records where its history will live instead. It is the single
// implementation behind `dacli new`, `dacli init` and `dacli adopt`, so every
// way of starting a dacli project produces the same arrangement — before, only
// greenfield `new` did it, which left the far more common case (adopting an
// existing repo) tracking its workspace on trunk.
//
// The two halves are ONE decision and are written in this order deliberately:
// gitignoring the workspace without a record branch does not tidy the
// bookkeeping, it deletes the history. So the config is written first, and if
// that fails nothing is ignored — a workspace is never left
// ignored-and-unrecorded.
//
// Why untracked is the right default: workspace records carried on the current
// branch fork with it. A task closed on a task branch is invisible on trunk
// until its record PR merges, so an orchestrator re-picks finished work; every
// git worktree checks out its own stale copy of the workspace it is supposed
// to SHARE; and trunk history fills with bookkeeping commits (in dacli's own
// repo, 251 of 429, one message repeated verbatim 61 times).
//
// It never rewrites a .gitignore it did not author: the entry is appended with
// a comment explaining where the history went, and a file that already
// excludes the workspace is left exactly as it is. Safe to call repeatedly.
//
// Returns whether it changed anything, so callers can report it without
// re-reading the file.
func UntrackFromTrunk(w *Workspace, repoRoot string) (changed bool, err error) {
	if err := w.SetRecordBranch(DefaultRecordBranch); err != nil {
		return false, fmt.Errorf("recording the workspace record branch: %w", err)
	}

	path := filepath.Join(repoRoot, ".gitignore")
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if HasIgnoreEntry(string(raw), Dir) {
		return false, nil
	}

	body := string(raw)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	if body != "" {
		body += "\n"
	}
	body += fmt.Sprintf(`# The dacli workspace lives on its own ref (%s), not on trunk: records
# carried on a branch fork with it, so a task closed on a task branch is
# invisible on trunk until its record merges. The history is NOT lost —
#   git log %s
#   git show %s:%s/projects/<p>/tasks/done/<file>
%s/
`, DefaultRecordBranch, DefaultRecordBranch, DefaultRecordBranch, Dir, Dir)

	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// HasIgnoreEntry reports whether body already excludes the named directory,
// matching any spelling that means the same thing — `.dacli`, `.dacli/`,
// `/.dacli`, `/.dacli/` — so a re-run or a hand-added entry never appends a
// duplicate. Comments and blank lines are skipped rather than matched.
func HasIgnoreEntry(body, dir string) bool {
	want := strings.Trim(dir, "/")
	for _, line := range strings.Split(body, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if strings.Trim(s, "/") == want {
			return true
		}
	}
	return false
}

// SetRecordBranch persists record_branch in the workspace config, appending it
// idempotently rather than rewriting a file dacli may not solely own. An
// existing value is never overwritten: an operator who chose a branch keeps
// it.
func (w *Workspace) SetRecordBranch(branch string) error {
	path := w.ConfigPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if k, _, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "record_branch" {
			return nil
		}
	}
	body := string(raw)
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	body += "record_branch: " + branch + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	w.RecordBranch = branch
	return nil
}
