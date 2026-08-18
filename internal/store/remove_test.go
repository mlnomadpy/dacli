package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func createRemovalRoleAndAgent(t *testing.T, w *workspace.Workspace, id string) {
	t.Helper()
	if err := CreateRole(w, "a-root", team.Role{Name: "fixer"}); err != nil {
		t.Fatal(err)
	}
	d := &mdstore.Doc{}
	d.Front.Set("id", id)
	d.Front.Set("kind", "agent")
	d.Front.Set("role", "fixer")
	d.Front.Set("grant", "rw")
	if err := mdstore.WriteFile(w.AgentPath(id), d); err != nil {
		t.Fatal(err)
	}
}

func writeRemovalRun(t *testing.T, w *workspace.Workspace, runID, child string, pid int) {
	t.Helper()
	path := filepath.Join(w.RunDir(runID), "proc.txt")
	if err := procmon.WriteRecord(path, procmon.Record{RunID: runID, Child: child, Role: "fixer", PID: pid, PGID: pid}); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveRoleUsesLiveAgentOccupancy(t *testing.T) {
	t.Run("finished holder does not block", func(t *testing.T) {
		w := removeWS(t)
		createRemovalRoleAndAgent(t, w, "a-fixer-done")
		writeRemovalRun(t, w, "01RUN-DONE", "a-fixer-done", 0)
		if err := RemoveRole(w, "fixer"); err != nil {
			t.Fatalf("RemoveRole with only a terminal holder: %v", err)
		}
		if _, err := os.Stat(w.AgentPath("a-fixer-done")); err != nil {
			t.Fatalf("historical agent record was removed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(w.RunDir("01RUN-DONE"), "proc.txt")); err != nil {
			t.Fatalf("historical run record was removed: %v", err)
		}
	})

	t.Run("live holder names agent and run", func(t *testing.T) {
		w := removeWS(t)
		createRemovalRoleAndAgent(t, w, "a-fixer-live")
		writeRemovalRun(t, w, "01RUN-LIVE", "a-fixer-live", os.Getpid())
		err := RemoveRole(w, "fixer")
		var ref ErrReferenced
		if !errors.As(err, &ref) {
			t.Fatalf("RemoveRole with live holder = %v, want ErrReferenced", err)
		}
		if got := ref.Error(); !strings.Contains(got, "a-fixer-live") || !strings.Contains(got, "01RUN-LIVE") {
			t.Fatalf("live refusal must name child and run: %v", err)
		}
	})

	t.Run("minted holder blocks until retired", func(t *testing.T) {
		w := removeWS(t)
		createRemovalRoleAndAgent(t, w, "a-fixer-new")
		if err := RemoveRole(w, "fixer"); err == nil {
			t.Fatal("minted-but-never-run holder did not block removal")
		}
		if err := RetireAgent(w, "a-fixer-new"); err != nil {
			t.Fatal(err)
		}
		if err := RemoveRole(w, "fixer"); err != nil {
			t.Fatalf("retired holder blocked removal: %v", err)
		}
	})
}

func TestRemoveRoleFailsClosedOnUnreadableState(t *testing.T) {
	t.Run("agent", func(t *testing.T) {
		w := removeWS(t)
		createRemovalRoleAndAgent(t, w, "a-fixer-bad")
		if err := os.RemoveAll(w.AgentsDir()); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(w.AgentsDir(), []byte("unreadable roster"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := RemoveRole(w, "fixer"); err == nil {
			t.Fatal("unreadable agent state certified removal")
		}
	})

	t.Run("run", func(t *testing.T) {
		w := removeWS(t)
		createRemovalRoleAndAgent(t, w, "a-fixer-bad")
		procPath := filepath.Join(w.RunDir("01RUN-BAD"), "proc.txt")
		if err := os.MkdirAll(procPath, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := RemoveRole(w, "fixer"); err == nil {
			t.Fatal("unreadable run state certified removal")
		}
	})
}

func removeWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "r")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// A shortcut is a command `dacli run` executes as the operator, so being
// unable to retract one by command is a standing capability. This is the
// highest-value inverse in the set (task 293).
func TestRemoveShortcut(t *testing.T) {
	w := removeWS(t)
	if err := CreateShortcut(w, "a-root", "deploy", "deploy it", "echo deploying", "read", nil, nil, ""); err != nil {
		t.Fatal(err)
	}
	if err := RemoveShortcut(w, "deploy"); err != nil {
		t.Fatalf("RemoveShortcut: %v", err)
	}
	if _, err := os.Stat(w.ShortcutPath("deploy")); !os.IsNotExist(err) {
		t.Errorf("shortcut file survived removal (stat err = %v)", err)
	}
	// Removing what is not there is not-found, never a silent success.
	var nf ErrNotFound
	if err := RemoveShortcut(w, "deploy"); !errors.As(err, &nf) {
		t.Errorf("removing an absent shortcut = %v, want ErrNotFound", err)
	}
}

// A dangling reference is worse than the mistake being removed: a role
// pointing at a deleted runtime fails at spawn time, far from the deletion
// that caused it, with an error that names neither.
func TestRemoveRuntimeRefusesWhileARoleRoutesToIt(t *testing.T) {
	w := removeWS(t)
	if err := CreateRuntime(w, "a-root", Runtime{Name: "rt", Binary: "/bin/sh", Mode: "stdin"}, ""); err != nil {
		t.Fatal(err)
	}
	if err := CreateRole(w, "a-root", team.Role{Name: "fixer", Runtime: "rt"}); err != nil {
		t.Fatal(err)
	}

	err := RemoveRuntime(w, "rt")
	var ref ErrReferenced
	if !errors.As(err, &ref) {
		t.Fatalf("RemoveRuntime = %v, want ErrReferenced while a role routes to it", err)
	}
	if len(ref.By) == 0 {
		t.Error("the refusal must name what still references it, not just refuse")
	}
	if _, serr := os.Stat(w.RuntimePath("rt")); serr != nil {
		t.Error("a refused removal must leave the object in place")
	}

	// Repoint the role and the removal proceeds.
	if err := RemoveRole(w, "fixer"); err != nil {
		t.Fatalf("RemoveRole: %v", err)
	}
	if err := RemoveRuntime(w, "rt"); err != nil {
		t.Errorf("RemoveRuntime after the referrer was gone: %v", err)
	}
}

// A traversing name must never resolve to a path outside the workspace, on the
// removal side as much as the creation side.
func TestRemoveRefusesTraversingNames(t *testing.T) {
	w := removeWS(t)
	for _, name := range []string{"../../escaped", "a/b"} {
		if err := RemoveShortcut(w, name); err == nil {
			t.Errorf("RemoveShortcut(%q) was accepted; a traversing name must be rejected", name)
		}
	}
}
