package store

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
)

// DefaultVersion is what a role or skill authored before versioning reads back
// as. Versions are a human-legible incrementing v1/v2, not semver: a role is a
// living capability bundle, not a released artifact, so a monotone counter is
// all the ordering anyone needs.
const DefaultVersion = "v1"

// Change is one git commit that touched a versioned file — the raw material of
// a changelog: who changed it, when, and what.
type Change struct {
	Hash    string
	Author  string
	When    string // human-relative ("3 days ago"), from git's %ar
	Subject string
}

// FileVersion reads the `version:` frontmatter of a role or skill file,
// defaulting to DefaultVersion for files that predate versioning. A malformed
// or unreadable file still yields the default rather than an error: a missing
// version must never block `show`.
func FileVersion(path string) string {
	d, err := mdstore.ReadFile(path)
	if err != nil {
		return DefaultVersion
	}
	if v, ok := d.Front.Get("version"); ok && v != "" {
		return v
	}
	return DefaultVersion
}

var trailingDigits = regexp.MustCompile(`([0-9]+)$`)

// NextVersion increments the trailing integer of a version string: v1→v2,
// v9→v10, 3→4. A value with no trailing number becomes v1, so a hand-edited or
// empty version bumps to a well-formed starting point rather than an error.
func NextVersion(cur string) string {
	m := trailingDigits.FindStringSubmatchIndex(cur)
	if m == nil {
		return DefaultVersion
	}
	prefix, num := cur[:m[2]], cur[m[2]:m[3]]
	n, err := strconv.Atoi(num)
	if err != nil {
		return DefaultVersion
	}
	return prefix + strconv.Itoa(n+1)
}

// BumpFileVersion rewrites the `version:` frontmatter to the next version,
// returning the old and new values. The file is rewritten in place; the caller
// commits it (through `dacli commit`) so the bump is attributed and lands in
// the very git history the changelog reads back.
func BumpFileVersion(path string) (old, next string, err error) {
	d, err := mdstore.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	old = DefaultVersion
	if v, ok := d.Front.Get("version"); ok && v != "" {
		old = v
	}
	next = NextVersion(old)
	d.Front.Set("version", next)
	if err := mdstore.WriteFile(path, d); err != nil {
		return "", "", err
	}
	return old, next, nil
}

// FileChangelog returns up to limit commits that touched path, newest first,
// derived straight from git history (`git log`). It is informational: an
// untracked file, a non-repo, or a missing git binary yields a nil slice and a
// nil error, never a failure — a changelog you cannot build must not break the
// `show` it garnishes. The boolean reports whether git could be consulted at
// all, so callers can distinguish "no history yet" from "not versioned in git".
func FileChangelog(path string, limit int) ([]Change, bool) {
	dir := filepath.Dir(path)
	if _, err := exec.LookPath("git"); err != nil {
		return nil, false
	}
	// A unit separator keeps subjects with spaces or pipes intact.
	const sep = "\x1f"
	format := strings.Join([]string{"%h", "%an", "%ar", "%s"}, sep)
	args := []string{"-C", dir, "log", "--follow", "-n", strconv.Itoa(limit), "--format=" + format, "--", path}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		// git present but this path is untracked, or dir is not a repo: no
		// history, still not an error the caller should surface.
		return nil, true
	}
	var changes []Change
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, sep, 4)
		if len(parts) != 4 {
			continue
		}
		changes = append(changes, Change{Hash: parts[0], Author: parts[1], When: parts[2], Subject: parts[3]})
	}
	return changes, true
}

// VersionIsStale reports whether path has changed since its current version was
// set — the signal behind "prompts to bump". It is true when the working tree
// has uncommitted edits to the file, or when commits landed on the file after
// the one that introduced the current `version:` value. `since` counts those
// later commits. A file whose version was never committed (a brand-new,
// unstaged role) is not stale: there is nothing to bump past yet.
func VersionIsStale(path, version string) (stale bool, since int) {
	dir := filepath.Dir(path)
	if _, err := exec.LookPath("git"); err != nil {
		return false, 0
	}
	base := filepath.Base(path)
	// Uncommitted edits to the file are the loudest staleness signal.
	if out, err := exec.Command("git", "-C", dir, "status", "--porcelain", "--", base).Output(); err == nil {
		if strings.TrimSpace(string(out)) != "" {
			return true, 0
		}
	}
	// The commit that last touched the current `version:` line. -S counts
	// occurrences of the exact string, so the newest such commit is where this
	// value was written.
	verCommit, err := exec.Command("git", "-C", dir, "log", "-1", "--format=%H",
		"-S", "version: "+version, "--", base).Output()
	if err != nil {
		return false, 0
	}
	vc := strings.TrimSpace(string(verCommit))
	if vc == "" {
		return false, 0 // version never committed yet — nothing to be stale against
	}
	out, err := exec.Command("git", "-C", dir, "log", "--format=%H", vc+"..HEAD", "--", base).Output()
	if err != nil {
		return false, 0
	}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(l) != "" {
			since++
		}
	}
	return since > 0, since
}

// FormatChangelog renders a changelog as aligned, human-legible lines for
// `role show` / `skill show`. An empty log renders a single honest line rather
// than nothing, so the reader knows the file simply has no committed history
// yet.
func FormatChangelog(changes []Change, gitSeen bool) string {
	if len(changes) == 0 {
		if !gitSeen {
			return "  (git unavailable — no changelog)"
		}
		return "  (no committed history yet)"
	}
	var b strings.Builder
	for _, c := range changes {
		fmt.Fprintf(&b, "  %s  %-16s %-14s %s\n", c.Hash, truncate(c.Author, 16), c.When, c.Subject)
	}
	return strings.TrimRight(b.String(), "\n")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
