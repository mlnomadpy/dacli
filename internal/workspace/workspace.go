// Package workspace locates and lays out the .dacli directory.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/ulid"
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
	w := &Workspace{Root: root}
	raw, err := os.ReadFile(w.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("workspace at %s has no readable config: %w", root, err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "id":
			w.ID = v
		case "name":
			w.Name = v
		case "format":
			// Refuse to operate on a format newer than this build understands,
			// rather than corrupting a workspace written by a later dacli.
			if v != fmt.Sprint(FormatVersion) {
				return nil, fmt.Errorf("workspace format %s is newer than this build's %d; upgrade dacli", v, FormatVersion)
			}
		}
	}
	return w, nil
}

// Init creates a workspace at root, along with the root agent.
func Init(root, name string) (*Workspace, error) {
	dir := filepath.Join(root, Dir)
	if _, err := os.Stat(dir); err == nil {
		return nil, fmt.Errorf("workspace already exists at %s", dir)
	}
	w := &Workspace{Root: root, Name: name, ID: ulid.New()}

	for _, d := range []string{w.AgentsDir(), w.ProjectsDir(), w.QueuesDir(), w.EventsDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}

	cfg := fmt.Sprintf("id: %s\nname: %s\nformat: %d\n", w.ID, w.Name, FormatVersion)
	if err := os.WriteFile(w.ConfigPath(), []byte(cfg), 0o644); err != nil {
		return nil, err
	}

	// Transcripts can contain repository content that was fine in a working
	// tree and is not fine in a pushed branch; compiled skill output is a
	// regenerable projection. Neither belongs in git, and the workspace
	// should enforce that without relying on the user's root .gitignore.
	ignore := "runs/\nbuild/\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(ignore), 0o644); err != nil {
		return nil, err
	}

	// The root agent: the identity used when DACLI_AGENT is unset.
	root_ := &mdstore.Doc{}
	root_.Front.Set("id", "a-root")
	root_.Front.Set("kind", string(model.KindAgent))
	root_.Front.Set("created", time.Now().UTC().Format(time.RFC3339))
	root_.Front.Set("created_by", "a-root")
	root_.Front.Set("grant", string(model.GrantRW))
	root_.Front.Set("role", "root")
	root_.Sections = []mdstore.Section{{Level: 1, Title: "root", Content: "The agent that initialized this workspace.\n"}}
	if err := mdstore.WriteFile(w.AgentPath("a-root"), root_); err != nil {
		return nil, err
	}
	return w, nil
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

func (w *Workspace) RuntimesDir() string { return w.dacli("runtimes") }

// PromptsDir holds workspace overrides for the embedded prompt registry —
// same nearest-wins rule as templates. See docs/PROMPTS.md.
func (w *Workspace) PromptsDir() string { return w.dacli("prompts") }

// TemplatesDir holds vendored project templates, which win over the
// embedded defaults of the same name.
func (w *Workspace) TemplatesDir() string { return w.dacli("templates") }

// SkillsLibDir is the canonical skill library (SKILLS.md).
func (w *Workspace) SkillsLibDir() string { return w.dacli("skills") }

// BuildSkillsDir is compiled skill output — a gitignored, regenerable
// projection (init writes build/ into .dacli/.gitignore).
func (w *Workspace) BuildSkillsDir(runtime, role string) string {
	return w.dacli("build", "skills", runtime, role)
}

func (w *Workspace) RuntimePath(name string) string {
	return filepath.Join(w.RuntimesDir(), name+".md")
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

// RunsDir holds per-run records: recorded briefs, invocations, outcomes.
// Gitignored — transcripts can contain repository content that was fine in a
// working tree and is not fine in a pushed branch.
func (w *Workspace) RunsDir() string { return w.dacli("runs") }

func (w *Workspace) RunDir(id string) string {
	return filepath.Join(w.RunsDir(), id)
}
