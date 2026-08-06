// Package workspace locates and lays out the .dacli directory.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/ulid"
)

// Dir is the workspace directory name, placed at the project root and
// committed to the repo alongside the code it describes.
const Dir = ".dacli"

// worktreesSubdir is the .dacli subdirectory that holds isolated per-agent git
// worktrees. It is defined once so WorktreesDir (which builds the path) and
// mainWorktreeRoot (which reads it back to redirect a worktree agent to the
// shared root) can never disagree on where worktrees live — the redirect is
// only deterministic if both use the same segment.
const worktreesSubdir = "worktrees"

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

	// DefaultTemplate is the process `dacli init --template` recorded. A
	// `project add` with no --template falls back to it, so the init flag has a
	// real mechanical effect instead of being silently dropped. Empty means the
	// solo default (no gates).
	DefaultTemplate string

	// AttributionDomain and TrailerPrefix control how agent authorship is
	// stamped onto commits: the author email domain and the `<Prefix>-Agent:` /
	// `<Prefix>-Role:` / `<Prefix>-Task:` trailer family.
	//
	// They are configurable because they are the loudest fingerprint dacli
	// leaves on a repository. Every commit carrying `Dacli-Agent:` and an
	// author at `@agent.dacli` makes a corpus of generated repositories
	// trivially clusterable as same-origin, which is a real problem when the
	// repositories are meant to be varied rather than obviously mass-produced.
	// Empty means the built-in defaults, so an existing workspace is unchanged
	// (dacli 196).
	AttributionDomain string
	TrailerPrefix     string
}

// Attribution returns the effective author email domain (with a leading "@")
// and trailer prefix, applying the built-in defaults when unset.
func (w *Workspace) Attribution() (domain, prefix string) {
	domain, prefix = w.AttributionDomain, w.TrailerPrefix
	if domain == "" {
		domain = "agent.dacli"
	}
	if !strings.HasPrefix(domain, "@") {
		domain = "@" + domain
	}
	if prefix == "" {
		prefix = "Dacli"
	}
	return domain, prefix
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
			// If dir is a linked worktree dacli created, the real workspace lives
			// in the MAIN root's .dacli. A worktree checks out a git-tracked
			// .dacli snapshot that is stale the moment the branch was cut, so
			// resolving there gives a spawned agent a shadow workspace: it can't
			// see its own freshly-minted identity or an uncommitted task, breaking
			// self-commit attribution and `task check`. Redirect to the shared
			// root so every agent shares ONE workspace (the append-only event
			// log makes concurrent writes safe). The redirect is deterministic and
			// git-free for dacli's own worktrees — see mainWorktreeRoot — so it
			// holds even where git is unavailable to the agent (task 296).
			if main := mainWorktreeRoot(dir); main != "" && main != dir {
				if fi, err := os.Stat(filepath.Join(main, Dir)); err == nil && fi.IsDir() {
					return open(main)
				}
			}
			return open(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNotFound
		}
		dir = parent
	}
}

// mainWorktreeRoot returns the shared root a linked-worktree .dacli at dir
// belongs to, or "" when dir is not a worktree dacli created (the main
// worktree, or no match).
//
// PATH detection is primary and git-free: dacli always creates a worktree at
// <root>/.dacli/worktrees/<name> (WorktreePath), so a .dacli found there has a
// deterministic shared root — the path segment before /.dacli/worktrees/ — with
// no subprocess at all. This is what lets a worktree agent resolve its identity
// even when git is unavailable to it or too old for --path-format=absolute; a
// silent git failure used to drop resolution back to the stale worktree
// snapshot and surface as a cryptic "agent token not recognized" (task 296).
//
// The git query is a FALLBACK for a worktree created by hand OUTSIDE
// .dacli/worktrees/, where the path carries no marker. It reads git's common
// dir — shared across all worktrees — whose parent is the main root; for the
// main worktree that parent is dir itself, so callers get "".
func mainWorktreeRoot(dir string) string {
	if root := rootFromWorktreePath(dir); root != "" {
		return root
	}
	out, err := gitx.Run(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return ""
	}
	common := strings.TrimSpace(out)
	if common == "" {
		return ""
	}
	return filepath.Dir(common)
}

// rootFromWorktreePath returns the shared root when dir sits under a
// <root>/.dacli/worktrees/ that dacli created, or "" otherwise. It matches the
// marker as a whole path so a repo that merely happens to contain the substring
// elsewhere is not mistaken for a worktree; the returned root is everything
// before the marker.
func rootFromWorktreePath(dir string) string {
	sep := string(filepath.Separator)
	marker := sep + Dir + sep + worktreesSubdir + sep
	i := strings.Index(dir, marker)
	if i <= 0 {
		return ""
	}
	return dir[:i]
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
		case "default_template":
			w.DefaultTemplate = v
		case "attribution_domain":
			w.AttributionDomain = v
		case "trailer_prefix":
			w.TrailerPrefix = v
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
	ignore := "runs/\nbuild/\nworktrees/\n"
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
	base := w.ProjectsDir()
	// Containment: a project slug is a single path segment. A slug carrying
	// `..`, a path separator, or an absolute path (from an explicit --slug or a
	// forged --project) is redirected to an in-workspace sentinel that never
	// exists, so the operation fails safely inside .dacli instead of reading or
	// writing outside it. Well-formed slugs ([a-z0-9-], via Slugify) never trip
	// this. See SafeSegment for the shared predicate.
	if !SafeSegment(slug) {
		return filepath.Join(base, "__invalid_segment__")
	}
	return filepath.Join(base, slug)
}

// SafeSegment reports whether s is usable as a single path component inside the
// workspace: non-empty, no separator, no `..`, not absolute, and not a lone
// dot. It is the guard for every user-supplied name that becomes a filesystem
// path (project slugs today; extend to role/runtime/queue names as needed).
func SafeSegment(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	if filepath.IsAbs(s) || strings.ContainsRune(s, '/') || strings.ContainsRune(s, filepath.Separator) {
		return false
	}
	return !strings.Contains(s, "..")
}

// SafeRelPath reports whether p is a relative path that stays inside the tree
// it is joined to. Unlike SafeSegment it permits separators — a gate's
// `artifact:` predicate legitimately names `internal/api/server.go` — but it
// still rejects absolute paths and any `..` traversal.
func SafeRelPath(p string) bool {
	if p == "" || filepath.IsAbs(p) {
		return false
	}
	clean := filepath.Clean(filepath.ToSlash(p))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	return !strings.HasPrefix(clean, "/")
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

// WorktreesDir holds isolated per-agent git worktrees for parallel work.
// Gitignored — they are working copies, not workspace state.
func (w *Workspace) WorktreesDir() string { return w.dacli(worktreesSubdir) }

// WorktreePath is keyed on project + seq + slug, not the slug alone: two tasks
// with the same title share a slug, and across projects even the seq can repeat,
// so a slug-only key made same-titled tasks share one worktree and commit onto
// the wrong branch (dacli 215). The name mirrors the branch (dacli/NNN-slug)
// with a project prefix so the on-disk layout stays greppable per project.
func (w *Workspace) WorktreePath(project string, seq int, slug string) string {
	name := fmt.Sprintf("%03d-%s", seq, slug)
	if project != "" {
		name = project + "-" + name
	}
	return filepath.Join(w.WorktreesDir(), name)
}

func (w *Workspace) RunDir(id string) string {
	return filepath.Join(w.RunsDir(), id)
}
