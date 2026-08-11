package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func gitFI(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func fiRepo(t *testing.T) *workspace.Workspace {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitFI(t, dir, "init", "-q")
	gitFI(t, dir, "config", "user.email", "x@x")
	gitFI(t, dir, "config", "user.name", "x")
	gitFI(t, dir, "checkout", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitFI(t, dir, "add", "-A")
	gitFI(t, dir, "commit", "-q", "-m", "base")
	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// FAILURE INJECTION at a real git call site.
//
// The landing check is the gate that decides whether a close records work the
// trunk actually received — the guard issue #382 exists because of. A git
// failure inside it must produce UNKNOWN, never a verdict: reporting "landed"
// because the query failed is the record-disagrees-with-reality class, and
// reporting "unlanded" would refuse work that may well have landed.
//
// Injected by asking about a trunk that does not exist and a sha that is not a
// commit — genuine git failures at the real call site, not mocked errors.
func TestLandingCheckReportsUnknownWhenGitCannotAnswer(t *testing.T) {
	w := fiRepo(t)
	head := strings.TrimSpace(runOut(t, w.Root, "rev-parse", "HEAD"))

	// Sanity first, or the assertions below could pass on a check that never
	// worked at all.
	if got := LandingOfRef(w, head, "main"); got != LandingLanded {
		t.Fatalf("baseline: HEAD must read as landed on main, got %v", got)
	}

	for _, tc := range []struct{ name, sha, trunk string }{
		{"trunk does not exist", head, "no-such-trunk"},
		{"sha is not a commit", "0000000000000000000000000000000000000000", "main"},
		{"empty trunk", head, ""},
		{"empty sha", "", "main"},
	} {
		if got := LandingOfRef(w, tc.sha, tc.trunk); got == LandingLanded {
			t.Errorf("%s: reported LANDED from a git query that could not answer", tc.name)
		}
	}
}

// A repository that is not a repository at all. Every git call fails, and the
// check must still return rather than panic or hang — the loop calls this per
// task, per cycle.
func TestLandingCheckSurvivesANonRepository(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "x") // no git init
	if err != nil {
		t.Fatal(err)
	}
	if got := LandingOfRef(w, "deadbeef", "main"); got == LandingLanded {
		t.Errorf("a non-repository reported LANDED, got %v", got)
	}
	if got := TrunkBranch(w); got != "" && got != "main" && got != "master" {
		t.Errorf("TrunkBranch on a non-repository returned %q", got)
	}
}

func runOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := c.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}
