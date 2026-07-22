package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
)

func TestNextVersion(t *testing.T) {
	cases := map[string]string{
		"v1":  "v2",
		"v9":  "v10",
		"v10": "v11",
		"3":   "4",
		"":    DefaultVersion, // no trailing digits → well-formed start
		"vX":  DefaultVersion,
	}
	for in, want := range cases {
		if got := NextVersion(in); got != want {
			t.Errorf("NextVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func writeVersioned(t *testing.T, path, version string) {
	t.Helper()
	d := &mdstore.Doc{}
	d.Front.Set("name", "x")
	if version != "" {
		d.Front.Set("version", version)
	}
	if err := mdstore.WriteFile(path, d); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestFileVersionAndBump(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "role.md")

	// A file with no version reads back the default.
	writeVersioned(t, path, "")
	if got := FileVersion(path); got != DefaultVersion {
		t.Fatalf("FileVersion(no version) = %q, want %q", got, DefaultVersion)
	}

	// Bump rewrites the frontmatter and returns old/new.
	writeVersioned(t, path, "v1")
	old, next, err := BumpFileVersion(path)
	if err != nil {
		t.Fatalf("bump: %v", err)
	}
	if old != "v1" || next != "v2" {
		t.Fatalf("bump = (%q, %q), want (v1, v2)", old, next)
	}
	if got := FileVersion(path); got != "v2" {
		t.Fatalf("FileVersion after bump = %q, want v2", got)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=t@example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestFileChangelogAndStaleness(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	path := filepath.Join(dir, "role.md")

	// v1, committed.
	writeVersioned(t, path, "v1")
	git(t, dir, "add", "role.md")
	git(t, dir, "commit", "-q", "-m", "add role v1")

	// A later content change while still v1 → stale, one commit past the version.
	writeVersioned2(t, path, "v1", "changed")
	git(t, dir, "commit", "-q", "-am", "tweak role")

	changes, seen := FileChangelog(path, 10)
	if !seen {
		t.Fatal("git should have been consulted")
	}
	if len(changes) != 2 {
		t.Fatalf("changelog len = %d, want 2", len(changes))
	}
	if changes[0].Subject != "tweak role" || changes[0].Author != "Tester" {
		t.Fatalf("newest change = %+v", changes[0])
	}

	stale, since := VersionIsStale(path, "v1")
	if !stale || since != 1 {
		t.Fatalf("VersionIsStale(v1) = (%v, %d), want (true, 1)", stale, since)
	}

	// Bump to v2 and commit → no longer stale.
	if _, _, err := BumpFileVersion(path); err != nil {
		t.Fatalf("bump: %v", err)
	}
	git(t, dir, "commit", "-q", "-am", "bump to v2")
	if stale, _ := VersionIsStale(path, "v2"); stale {
		t.Fatal("VersionIsStale(v2) = true right after committing the bump")
	}
}

// writeVersioned2 writes a versioned doc carrying an extra summary so the file
// content genuinely differs between commits.
func writeVersioned2(t *testing.T, path, version, summary string) {
	t.Helper()
	d := &mdstore.Doc{}
	d.Front.Set("name", "x")
	d.Front.Set("version", version)
	d.Front.Set("summary", summary)
	if err := mdstore.WriteFile(path, d); err != nil {
		t.Fatalf("write: %v", err)
	}
}
