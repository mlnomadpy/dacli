package store

import (
	"fmt"
	"os"
	"strings"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

// --- Removal inverses.
//
// Ten object types could be created by a command and removed only by hand-
// editing markdown, so a mistake a command made needed a human with a text
// editor to undo (task 293). That is worst for the two that define EXECUTION:
// a shortcut is a command `dacli run` executes, and a runtime names the binary
// and allowlist every child launches under — being unable to retract either by
// command is a capability that only accumulates.
//
// Every removal here refuses while something still points at the object. A
// dangling reference is worse than the mistake being removed: a role pointing
// at a deleted runtime fails at spawn time, far from the deletion that caused
// it, and the error names neither.
//
// Deliberately NOT given inverses: notes (findings and decisions) and risks.
// Those are the record — the append-only trail the whole tool exists to keep —
// and a finding that can be deleted is a finding a future reader cannot trust.
// Correct them by filing a superseding note, which leaves both visible. This
// is a decision, not an omission.

// ErrReferenced reports that an object cannot be removed because something
// still names it. It carries the referrers so the caller can print them.
type ErrReferenced struct {
	Kind, Name string
	By         []string
}

func (e ErrReferenced) Error() string {
	return fmt.Sprintf("%s %q is still referenced by %s", e.Kind, e.Name, strings.Join(e.By, ", "))
}

// RemoveRole deletes a role, refusing while any non-retired agent holds it.
// Removing a role out from under a live agent would leave its work
// unattributable to any declared scope or grant.
func RemoveRole(w *workspace.Workspace, name string) error {
	var by []string
	if agents, err := ListAgents(w); err == nil {
		for _, a := range agents {
			if a.Role == name && !a.Retired {
				by = append(by, "agent "+a.ID)
			}
		}
	}
	if len(by) > 0 {
		return ErrReferenced{Kind: "role", Name: name, By: by}
	}
	return removeObject(w.RolePath(name), "role", name)
}

// RemoveRuntime deletes a runtime adapter, refusing while a role routes to it.
func RemoveRuntime(w *workspace.Workspace, name string) error {
	var by []string
	if roles, err := LoadRoles(w); err == nil {
		for _, r := range roles {
			if r.Runtime == name {
				by = append(by, "role "+r.Name)
			}
		}
	}
	if len(by) > 0 {
		return ErrReferenced{Kind: "runtime", Name: name, By: by}
	}
	return removeObject(w.RuntimePath(name), "runtime", name)
}

// RemoveShortcut deletes a shortcut. Nothing structurally references one, so
// there is no reference check — but this is the highest-value inverse in the
// set: a shortcut is a command `dacli run` executes as the operator, and an
// unremovable one is a standing capability.
func RemoveShortcut(w *workspace.Workspace, name string) error {
	return removeObject(w.ShortcutPath(name), "shortcut", name)
}

// RemoveQueue deletes a queue.
func RemoveQueue(w *workspace.Workspace, slug string) error {
	return removeObject(w.QueuePath(slug), "queue", slug)
}

// removeObject is the shared delete: refuse a traversing name, report a
// missing object as not-found (exit 4, never a silent success), and delete.
func removeObject(path, kind, name string) error {
	if !workspace.SafeSegment(name) {
		return fmt.Errorf("invalid %s name %q: must be a single path segment without '/' or '..'", kind, name)
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound{Ref: kind + "/" + name}
		}
		return err
	}
	return os.Remove(path)
}
