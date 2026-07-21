// Package workspace locates and lays out the .dacli directory.
package workspace

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/mlnomadpy/dacli/internal/model"
)

// Dir is the workspace directory name, placed at the project root and
// committed to the repo alongside the code it describes.
const Dir = ".dacli"

// FormatVersion is written to config.yml. 0 means pre-1.0: the on-disk format
// may still change. From 1 onward, changes are additive only.
const FormatVersion = 0

// ErrNotFound is returned when no workspace exists at or above the start path.
var ErrNotFound = errors.New("no dacli workspace found (run `dacli init`)")

// Workspace is an opened .dacli directory.
type Workspace struct {
	Root string // the project root; Root/.dacli exists
	Name string
	ID   string
}

// Find walks up from start looking for a .dacli directory, the same way git
// finds .git. This is what lets a subagent run dacli from any subdirectory.
func Find(start string) (*Workspace, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		if fi, err := os.Stat(filepath.Join(dir, Dir)); err == nil && fi.IsDir() {
			return open(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNotFound
		}
		dir = parent
	}
}

func open(root string) (*Workspace, error) {
	// TODO: read .dacli/config.yml for Name, ID, and format version; refuse to
	// operate on a format version newer than FormatVersion rather than
	// corrupting a workspace written by a later build.
	return &Workspace{Root: root}, nil
}

// Init creates a workspace at root, along with the root agent.
func Init(root, name string) (*Workspace, error) {
	// TODO: create config.yml, agents/, projects/, queues/, events/;
	// mint the root agent with GrantRW; refuse if .dacli already exists.
	return nil, errors.New("not implemented")
}

// --- Path layout. Every path in the workspace is derived here, so the layout
// is defined in exactly one place. ---

func (w *Workspace) dacli(parts ...string) string {
	return filepath.Join(append([]string{w.Root, Dir}, parts...)...)
}

func (w *Workspace) ConfigPath() string   { return w.dacli("config.yml") }
func (w *Workspace) AgentsDir() string    { return w.dacli("agents") }
func (w *Workspace) ProjectsDir() string  { return w.dacli("projects") }
func (w *Workspace) QueuesDir() string    { return w.dacli("queues") }
func (w *Workspace) EventsDir() string    { return w.dacli("events") }
func (w *Workspace) RolesDir() string     { return w.dacli("roles") }
func (w *Workspace) ShortcutsDir() string { return w.dacli("shortcuts") }

func (w *Workspace) RolePath(name string) string {
	return filepath.Join(w.RolesDir(), name+".md")
}

func (w *Workspace) ShortcutPath(name string) string {
	return filepath.Join(w.ShortcutsDir(), name+".md")
}

func (w *Workspace) AgentPath(id string) string {
	return filepath.Join(w.AgentsDir(), id+".md")
}

func (w *Workspace) ProjectDir(slug string) string {
	return filepath.Join(w.ProjectsDir(), slug)
}

func (w *Workspace) ProjectPath(slug string) string {
	return filepath.Join(w.ProjectDir(slug), "project.md")
}

// TasksDir returns the folder for a status. Status is folder position, so this
// is also the only way a task's status is ever set: by moving the file.
func (w *Workspace) TasksDir(project string, s model.Status) string {
	return filepath.Join(w.ProjectDir(project), "tasks", string(s))
}

func (w *Workspace) NotesDir(project string, k model.NoteKind) string {
	return filepath.Join(w.ProjectDir(project), "notes", noteFolder(k))
}

func noteFolder(k model.NoteKind) string {
	switch k {
	case model.NoteDecision:
		return "decisions"
	case model.NoteFinding:
		return "findings"
	case model.NoteMetric:
		return "metrics"
	default:
		return "refs"
	}
}

// RisksDir holds the project's impact x likelihood matrix entries.
func (w *Workspace) RisksDir(project string) string {
	return filepath.Join(w.ProjectDir(project), "risks")
}

// GlossaryPath is the project's shared term list, emitted into every brief.
func (w *Workspace) GlossaryPath(project string) string {
	return filepath.Join(w.ProjectDir(project), "glossary.md")
}

func (w *Workspace) QueuePath(slug string) string {
	return filepath.Join(w.QueuesDir(), slug+".md")
}

// EventPath returns the path for a new event. Named by ULID, so it sorts by
// creation time and two concurrent writers can never collide.
func (w *Workspace) EventPath(ts, ulid, agent string, kind model.EventKind) string {
	// ts is YYYY/MM/DD.
	return filepath.Join(w.EventsDir(), ts, ulid+"-"+agent+"-"+string(kind)+".md")
}
