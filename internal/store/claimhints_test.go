package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

// PathHints is deliberately crude — a slash or a .go suffix is enough — because
// for routing a spurious token costs one weak tie-break vote. The loop then
// reused it as `spawn --claim`, where a claim REFUSES every staged file outside
// it. Task 338's acceptance criteria mention the gosec rule list
// "G104/G301/G302/G306"; that became the agent's entire claim and its commit of
// eighteen legitimate files was refused (issue #427).
func TestClaimHintsRejectsProseThatMerelyContainsASlash(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "g", ""); err != nil {
		t.Fatal(err)
	}
	// A real file the task legitimately names.
	real := filepath.Join("internal", "store", "store.go")
	if err := os.MkdirAll(filepath.Join(w.Root, "internal", "store"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.Root, real), []byte("package store\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	task, err := CreateTask(w, "a-root", "core", "Stage gosec with a curated rule list", TaskOpts{
		Accept: []string{
			"gosec runs with G104/G301/G302/G306 excluded",
			"the and/or cases in internal/store/store.go are covered",
			"a 50/50 split is handled",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// The crude hinter still yields the junk — that is its documented contract
	// and routing tolerates it. If this stops being true the test below is
	// measuring nothing, so assert it.
	hints := task.PathHints()
	sawJunk := false
	for _, h := range hints {
		if h == "G104/G301/G302/G306" || h == "and/or" || h == "50/50" {
			sawJunk = true
		}
	}
	if !sawJunk {
		t.Fatalf("PathHints no longer yields the junk tokens this test guards against: %v", hints)
	}

	got := ClaimHints(w.Root, task)
	for _, bad := range []string{"G104/G301/G302/G306", "and/or", "50/50"} {
		for _, g := range got {
			if g == bad {
				t.Errorf("%q became a claim; prose containing a slash is not a path", bad)
			}
		}
	}
	// The real path survives, or the filter has thrown away the signal too.
	found := false
	for _, g := range got {
		if g == real || g == "internal/store/store.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("the real path %q was filtered out along with the junk: %v", real, got)
	}
}

// A bare filename is plausible and still wrong: claims match by exact path or
// path-prefix (procmon.PathsOverlap), so a claim of "acceptance.go" overlaps NO
// staged file and blocks the whole commit just as thoroughly as junk does.
//
// Yielding nothing is the safe outcome: an agent with no recorded claim is
// warned once and proceeds, while a wrong claim is a lockout only --force gets
// past.
func TestClaimHintsDropsABareFilenameThatIsNotARepoPath(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "g", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(w.Root, "internal", "features", "acceptance"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.Root, "internal", "features", "acceptance", "acceptance.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	task, err := CreateTask(w, "a-root", "core", "Cover the unlanded-close refusal", TaskOpts{
		Accept: []string{"acceptance.go:148 refuses the close"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := ClaimHints(w.Root, task); len(got) != 0 {
		t.Errorf("ClaimHints = %v; a bare filename is not a repo path and would match nothing staged", got)
	}
}
