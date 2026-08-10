package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Every task/risk gate is "for every X, P(X)", which is vacuously TRUE on an
// empty slice. The store reads deliberately return an error rather than an
// empty set precisely so a caller cannot confuse the two — and evaluate()
// dropped it with `, _`, so a permission or I/O fault made every gate pass
// with ZERO tasks examined while `stage advance` printed "every gate passed".
func TestQuantifierGatesFailClosedOnAnUnreadableSet(t *testing.T) {
	// A workspace whose task directory cannot be read: the project exists in
	// the index but its tasks path is a FILE, so ReadDir fails with a
	// non-ENOENT error.
	w, p := unreadableTasksProject(t)

	for _, arg := range []string{"all_have_acceptance", "all_have_estimate", "musts_done"} {
		c := evaluate(w, p, Predicate{Kind: "tasks", Arg: arg})
		if c.OK {
			t.Errorf("%s passed on an unreadable task set — a gate must never certify what it could not read", arg)
		}
		if !strings.Contains(strings.ToLower(c.Why), "could not read") {
			t.Errorf("%s: Why = %q, want it to name the read failure", arg, c.Why)
		}
	}
}

// unreadableTasksProject builds a project whose tasks directory cannot be
// listed: a status folder is replaced by a regular FILE, so ReadDir returns a
// non-ENOENT error — the transient-fault shape the gates must not read as
// "there are no tasks".
func unreadableTasksProject(t *testing.T) (*workspace.Workspace, *store.Project) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.CreateProject(w, agentid.RootID, "Core", "core", "", "")
	if err != nil {
		t.Fatal(err)
	}
	open := w.TasksDir("core", "open")
	if err := os.RemoveAll(open); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(open), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(open, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	return w, p
}
