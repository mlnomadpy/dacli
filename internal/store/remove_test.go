package store

import (
	"errors"
	"os"
	"testing"

	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

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
