// Containment: nothing a caller names may write outside .dacli.
//
// Every object here is stored as <name>.md under a fixed directory, so the
// name IS a path component and a traversing one escapes the workspace. The
// guard is one line, which is exactly why it drifts: CreateQueue, CreateRole
// and CreateProject carried it while CreateShortcut and CreateRuntime did not,
// and the shortcut gap was asymmetric on top of that — RemoveShortcut refuses
// a traversing name, so a file created that way could not be deleted by the
// tool that wrote it.
//
// This is the same failure mode as every other by-convention guard in this
// codebase (Flags.Reject reached 4 handlers of 112). The countermeasure is the
// arch_test.go one: enumerate the constructors HERE, so adding a new one
// without a guard fails a test rather than waiting to be noticed.
package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// traversingNames are the shapes that reach outside a single path segment.
// Each is tried against every constructor below.
var traversingNames = []string{
	"../escaped",
	"../../escaped",
	"a/b",
	"..",
	"sub/../../escaped",
}

// TestConstructorsRefuseTraversingNames drives every store constructor whose
// caller-supplied name becomes a filename.
func TestConstructorsRefuseTraversingNames(t *testing.T) {
	// One entry per constructor. A new constructor that takes a name and does
	// not appear here is the thing this test cannot catch, so keep it complete
	// — the sibling check below fails if a name-taking Create* is missing.
	ctors := []struct {
		name string
		call func(w *workspace.Workspace, bad string) error
	}{
		{"CreateQueue", func(w *workspace.Workspace, bad string) error {
			_, err := CreateQueue(w, "a-root", bad, "T", []string{"step one"})
			return err
		}},
		{"CreateRole", func(w *workspace.Workspace, bad string) error {
			return CreateRole(w, "a-root", team.Role{Name: bad, Grant: "ro"})
		}},
		{"CreateProject", func(w *workspace.Workspace, bad string) error {
			_, err := CreateProject(w, "a-root", "T", bad, "goal", "")
			return err
		}},
		{"CreateShortcut", func(w *workspace.Workspace, bad string) error {
			return CreateShortcut(w, "a-root", bad, "s", "echo hi", "read", nil, nil, "why")
		}},
		{"CreateRuntime", func(w *workspace.Workspace, bad string) error {
			return CreateRuntime(w, "a-root", Runtime{Name: bad, Binary: "echo"}, "note")
		}},
	}

	for _, c := range ctors {
		for _, bad := range traversingNames {
			t.Run(c.name+"/"+bad, func(t *testing.T) {
				w := runtimeWorkspace(t)
				// A witness OUTSIDE .dacli: if the constructor escaped, the
				// file lands somewhere under the repo root but not under
				// .dacli, and the walk below finds it. This checks the actual
				// containment rather than trusting the error text.
				before := filesOutsideDacli(t, w)

				err := c.call(w, bad)
				if err == nil {
					t.Fatalf("%s accepted the traversing name %q", c.name, bad)
				}
				if !strings.Contains(err.Error(), "path segment") {
					t.Fatalf("%s refused %q but not as a containment fault: %v", c.name, bad, err)
				}
				if after := filesOutsideDacli(t, w); len(after) != len(before) {
					t.Fatalf("%s wrote outside .dacli given %q: %v", c.name, bad, diffStrings(before, after))
				}
			})
		}
	}
}

// TestProjectScopedConstructorsRefuseATraversingProject pins the containment
// that CreateRisk and CreateNote get INDIRECTLY: neither guards the project
// name itself, but both require the project to load first, and a traversing
// project resolves to no project.md. That is real containment, and it is worth
// a test precisely because it is a side effect of a lookup rather than a
// stated guard — a future change that made the project optional would open the
// path silently.
func TestProjectScopedConstructorsRefuseATraversingProject(t *testing.T) {
	for _, bad := range traversingNames {
		w := runtimeWorkspace(t)
		before := filesOutsideDacli(t, w)

		if _, err := CreateRisk(w, "a-root", bad, "A risk", "high", "high", nil, ""); err == nil {
			t.Fatalf("CreateRisk accepted traversing project %q", bad)
		}
		if _, err := CreateNote(w, "a-root", bad, "finding", "A note", NoteOpts{}); err == nil {
			t.Fatalf("CreateNote accepted traversing project %q", bad)
		}
		if _, err := CreateTask(w, "a-root", bad, "A task", TaskOpts{Accept: []string{"a"}}); err == nil {
			t.Fatalf("CreateTask accepted traversing project %q", bad)
		}
		if after := filesOutsideDacli(t, w); len(after) != len(before) {
			t.Fatalf("a project-scoped constructor wrote outside .dacli given %q: %v", bad, diffStrings(before, after))
		}
	}
}

// filesOutsideDacli walks the workspace root and returns every file that is not
// under .dacli — the direct measurement of "did anything escape".
func filesOutsideDacli(t *testing.T, w *workspace.Workspace) []string {
	t.Helper()
	dacli := filepath.Join(w.Root, workspace.Dir)
	var out []string
	// The parent of the temp root too: "../escaped" lands there, not under it.
	for _, root := range []string{w.Root, filepath.Dir(w.Root)} {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil //nolint:nilerr // an unreadable entry is not an escape
			}
			if strings.HasPrefix(path, dacli+string(os.PathSeparator)) {
				return nil
			}
			out = append(out, path)
			return nil
		})
	}
	return out
}

func diffStrings(before, after []string) []string {
	seen := map[string]bool{}
	for _, b := range before {
		seen[b] = true
	}
	var added []string
	for _, a := range after {
		if !seen[a] {
			added = append(added, a)
		}
	}
	return added
}
