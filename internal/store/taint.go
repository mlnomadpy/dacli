package store

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// TaintHit is one artifact carrying a suspect origin.
type TaintHit struct {
	Kind    string // event | note
	ID      string
	Actor   string
	Origin  string
	About   string // task ref the artifact attaches to
	Project string // for notes
	Path    string
}

// TaintResult is the blast radius of a source: everything that carried its
// provenance, and every brief that would therefore surface it.
type TaintResult struct {
	Source   string
	Hits     []TaintHit
	Tasks    map[string]bool // task refs directly attached to a hit
	Projects map[string]bool // projects whose briefs surface a hit's finding
	TreeWide bool            // a workspace-scoped hit reaches EVERY project's briefs
}

// Taint walks provenance forward from a suspect source (an origin string like
// "file:cron/settle.go" or "external:some-user", matched as a substring so
// "file:" catches every file). It does NOT prevent injection — nothing here
// does; it converts "attribution helps a human audit afterward" from a
// sentence into a command (PROPOSALS P4). A finding note reaches every brief
// in its project, so a single hit taints a whole project's context.
func Taint(w *workspace.Workspace, source string) (*TaintResult, error) {
	res := &TaintResult{Source: source, Tasks: map[string]bool{}, Projects: map[string]bool{}}
	// Case-insensitive substring match (reviewer F3): file:Configs/Evil.yml
	// must not evade `taint file:configs/evil.yml` on a case-folding FS.
	needle := strings.ToLower(source)
	matches := func(origin string) bool {
		return origin != "" && strings.Contains(strings.ToLower(origin), needle)
	}

	// Events carrying the origin. PENDING only (reviewer F5): an applied
	// event has become a note that this walk also counts — walking both
	// double-counts every synced finding.
	_ = filepath.WalkDir(w.EventsDir(), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		doc, err := mdstore.ReadFile(path)
		if err != nil {
			return nil
		}
		if applied, _ := doc.Front.Get("applied"); applied == "true" {
			return nil
		}
		origin, _ := doc.Front.Get("origin")
		if !matches(origin) {
			return nil
		}
		hit := TaintHit{Kind: "event", Origin: origin, Path: path}
		hit.ID, _ = doc.Front.Get("id")
		hit.Actor, _ = doc.Front.Get("created_by")
		if a, ok := doc.Front.Get("about"); ok {
			hit.About = canonRef(w, strings.TrimSuffix(strings.TrimPrefix(a, "[["), "]]"))
			res.Tasks[hit.About] = true
		}
		res.Hits = append(res.Hits, hit)
		return nil
	})

	// Notes carrying the origin. Every note kind, metrics included (reviewer
	// F2). A note reaches its whole project's briefs; a workspace-scoped note
	// reaches EVERY project's briefs (reviewer F1: it surfaces tree-wide as a
	// "Lesson from other projects", regardless of note kind).
	projects, _ := ListProjects(w)
	for _, p := range projects {
		for _, kind := range []model.NoteKind{model.NoteFinding, model.NoteDecision, model.NoteRef, model.NoteMetric} {
			docs, _ := ListNotes(w, p.Slug, kind)
			for _, doc := range docs {
				origin, _ := doc.Front.Get("origin")
				if !matches(origin) {
					continue
				}
				hit := TaintHit{Kind: "note", Origin: origin, Project: p.Slug}
				hit.ID, _ = doc.Front.Get("id")
				hit.Actor, _ = doc.Front.Get("created_by")
				if a, ok := doc.Front.Get("about"); ok {
					hit.About = canonRef(w, strings.TrimSuffix(strings.TrimPrefix(a, "[["), "]]"))
					res.Tasks[hit.About] = true
				}
				res.Hits = append(res.Hits, hit)

				scope, _ := doc.Front.Get("scope")
				if scope == "workspace" {
					res.TreeWide = true // reaches every project's briefs
				} else if kind == model.NoteFinding {
					res.Projects[p.Slug] = true // findings surface project-wide
				}
			}
		}
	}
	return res, nil
}

// canonRef resolves any task ref (ULID, seq, slug) to its slug, so the
// exposed-brief set does not list one task twice under two labels
// (reviewer F5).
func canonRef(w *workspace.Workspace, ref string) string {
	if t, err := FindTask(w, ref); err == nil {
		return t.Slug
	}
	return ref
}

// ExposedBriefs returns the task refs whose briefs would surface a tainted
// artifact: every task in a tainted project (a finding note reaches all of
// them) plus any directly-attached task. This is the blast radius in the
// unit that matters — how many agents got fed the poison.
func (r *TaintResult) ExposedBriefs(w *workspace.Workspace) []string {
	exposed := map[string]bool{}
	for ref := range r.Tasks {
		exposed[ref] = true
	}
	// A workspace-scoped hit reaches every project; otherwise only the
	// projects a finding taints project-wide.
	projects := map[string]bool{}
	if r.TreeWide {
		for _, p := range mustProjects(w) {
			projects[p] = true
		}
	}
	for proj := range r.Projects {
		projects[proj] = true
	}
	for proj := range projects {
		tasks, _ := ListTasks(w, proj, "")
		for _, t := range tasks {
			exposed[t.Slug] = true
		}
	}
	out := make([]string, 0, len(exposed))
	for ref := range exposed {
		out = append(out, ref)
	}
	return out
}

func mustProjects(w *workspace.Workspace) []string {
	ps, _ := ListProjects(w)
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Slug
	}
	return out
}
