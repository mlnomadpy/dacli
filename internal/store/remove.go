package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"

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

// --- Task lifecycle inverses (dacli 340).
//
// A task could be closed but never reopened, and created but never removed. So
// a wrongly-closed task — the operator ran `accept --force` over a batch
// without reading what was in it, which is exactly how tasks 336 and 339 were
// falsely marked done — could only be corrected by editing the markdown store
// by hand. The tool's own record was the one thing it gave you no command to
// fix.
//
// Both refuse loudly rather than doing something plausible: ReopenTask will not
// touch a task that is already open, and RemoveTask will not delete one anything
// still points at.

// ReopenTask moves a done or blocked task back to open and UNCHECKS its
// acceptance boxes, because the boxes are a claim that the work was verified
// and a reopen says that claim was wrong. It returns how many boxes it cleared
// so the caller can report it — a silent unchecking would replace one false
// record with a different one.
func ReopenTask(w *workspace.Workspace, t *Task, actor, reason string) (int, error) {
	if t.Status == model.StatusOpen || t.Status == model.StatusActive {
		return 0, fmt.Errorf("%03d-%s is already %s — nothing to reopen", t.Seq, t.Slug, t.Status)
	}
	if strings.TrimSpace(reason) == "" {
		// A reopen with no reason is a mystery to the next reader, and the
		// whole point of the log is that a later reader can reconstruct why.
		return 0, fmt.Errorf("a reopen needs a reason: what makes the close wrong?")
	}
	cleared := UncheckAllAcceptance(t)
	AppendLog(t, fmt.Sprintf("reopened by %s: %s (cleared %d acceptance box(es) — the close claimed work that was not verified)", actor, reason, cleared))
	if err := SaveTask(t); err != nil {
		return 0, err
	}
	return cleared, MoveTask(w, t, model.StatusOpen)
}

// UncheckAllAcceptance clears every checked acceptance box IN PLACE and returns
// how many it cleared. The mirror of CheckAllAcceptance, and it preserves prose,
// blank lines and nested indentation for the same reason (dacli 335).
func UncheckAllAcceptance(t *Task) int {
	sec, ok := t.Doc.Section("Acceptance")
	if !ok {
		return 0
	}
	lines := strings.Split(sec.Content, "\n")
	cleared := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if !strings.HasPrefix(lower, "- [x]") {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		lines[i] = indent + "- [ ]" + trimmed[len("- [x]"):]
		cleared++
	}
	t.Doc.SetSection("Acceptance", strings.Join(lines, "\n"))
	return cleared
}

// RemoveTask deletes a task file outright, for the case a task should never
// have existed — a probe, a duplicate, a mis-filed note. It refuses while
// anything still points at the task, because a dangling reference fails far
// from the deletion that caused it and names neither.
//
// Deliberately NOT a way to retract real work: a task with history is corrected
// by reopening it, which leaves the record visible. Removal is for tasks whose
// existence was the mistake.
func RemoveTask(w *workspace.Workspace, t *Task) error {
	// A LIVE agent holding this task outranks everything below. Removing it
	// leaves that agent working a ref that now resolves to nothing — or worse,
	// to a DIFFERENT task, because a freed seq is handed out again. Reported
	// from inside the failure: an estimator mid-investigation watched
	// `dacli task show 344` start returning someone else's finished task,
	// with no signal that its own had been deleted (issue #433). This is not
	// --force-able; stop the agent first, because the alternative is a run
	// that cannot be made correct.
	if held := liveClaimants(w, t); len(held) > 0 {
		return ErrReferenced{
			Kind: "task",
			Name: fmt.Sprintf("%03d-%s", t.Seq, t.Slug),
			By:   held,
		}
	}
	if by := taskReferrers(w, t); len(by) > 0 {
		return ErrReferenced{Kind: "task", Name: fmt.Sprintf("%03d-%s", t.Seq, t.Slug), By: by}
	}
	return os.Remove(t.Path)
}

// liveClaimants names the agents whose run is still alive and whose run record
// points at this task.
//
// aboutRefs deliberately searches only the RECORD — events and notes — because
// that is what "something references this" meant when RemoveTask was written.
// It never looked at .dacli/runs, so a task could be referenced by a running
// process and read as unreferenced. The sibling condition, missed (issue #433).
func liveClaimants(w *workspace.Workspace, t *Task) []string {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return nil
	}
	live := liveChildren(w)
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec, rerr := procmon.ReadRecord(filepath.Join(w.RunDir(e.Name()), "proc.txt"))
		if rerr != nil || rec.Child == "" || !live[rec.Child] {
			continue
		}
		if rec.Task == t.ID || rec.Task == fmt.Sprintf("%03d", t.Seq) {
			out = append(out, fmt.Sprintf("live agent %s (run %s) is working it — stop it first (`dacli kill %s`) or let it finish", rec.Child, rec.RunID, rec.RunID))
		}
	}
	return out
}

// taskReferrers lists what still names this task: another task's dependency,
// or a note or event filed about it.
func taskReferrers(w *workspace.Workspace, t *Task) []string {
	var by []string
	all, _ := ListTasks(w, "", "")
	for _, other := range all {
		if other.ID == t.ID {
			continue
		}
		for _, d := range other.Deps() {
			if d.Ref == t.ID || d.Ref == fmt.Sprintf("%03d", t.Seq) {
				by = append(by, fmt.Sprintf("task %03d-%s depends on it", other.Seq, other.Slug))
			}
		}
	}
	if t.ID != "" {
		if hits := aboutRefs(w, t.ID); len(hits) > 0 {
			by = append(by, hits...)
		}
	}
	return by
}

// aboutRefs finds the notes and events whose `about` names this id.
//
// Scoped to the WORKSPACE's own event and note directories, never the whole
// repository. An earlier draft walked w.Root as a fallback, which read every
// .md file in the tree — slow on a real repo, and capable of matching an id
// inside a document that has nothing to do with the workspace. What counts as
// a reference is a dacli record, so only dacli's records are searched.
func aboutRefs(w *workspace.Workspace, id string) []string {
	roots := []string{w.EventsDir()}
	if ps, err := ListProjects(w); err == nil {
		for _, p := range ps {
			for _, k := range []model.NoteKind{model.NoteFinding, model.NoteDecision, model.NoteMetric, model.NoteRef} {
				roots = append(roots, w.NotesDir(p.Slug, k))
			}
		}
	}
	needle := "[[" + id + "]]"
	var out []string
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			//nolint:nilerr // fs.WalkDirFunc: nil skips this entry and keeps walking
			if err != nil {
				return nil
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			// G122 (symlink TOCTOU in a walk callback) is accepted here: the
			// tree being walked is the workspace's own record directory, whose
			// contents dacli writes. Anyone able to plant a symlink inside it
			// already has write access to every task file this check protects.
			// os.Root would close it properly and needs Go 1.24; this module
			// targets 1.22.
			raw, rerr := os.ReadFile(path) //nolint:gosec // G122: see above
			if rerr != nil {
				//nolint:nilerr // an unreadable file is not a reference; keep walking
				return nil
			}
			if strings.Contains(string(raw), needle) {
				out = append(out, "recorded in "+filepath.Base(path))
			}
			return nil
		})
	}
	return out
}
