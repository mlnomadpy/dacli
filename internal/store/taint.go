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
}

// Taint walks provenance forward from a suspect source (an origin string like
// "file:cron/settle.go" or "external:some-user", matched as a substring so
// "file:" catches every file). It does NOT prevent injection — nothing here
// does; it converts "attribution helps a human audit afterward" from a
// sentence into a command (PROPOSALS P4). A finding note reaches every brief
// in its project, so a single hit taints a whole project's context.
func Taint(w *workspace.Workspace, source string) (*TaintResult, error) {
	res := &TaintResult{Source: source, Tasks: map[string]bool{}, Projects: map[string]bool{}}

	// Events carrying the origin.
	_ = filepath.WalkDir(w.EventsDir(), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		doc, err := mdstore.ReadFile(path)
		if err != nil {
			return nil
		}
		origin, _ := doc.Front.Get("origin")
		if origin == "" || !strings.Contains(origin, source) {
			return nil
		}
		hit := TaintHit{Kind: "event", Origin: origin, Path: path}
		hit.ID, _ = doc.Front.Get("id")
		hit.Actor, _ = doc.Front.Get("created_by")
		if a, ok := doc.Front.Get("about"); ok {
			hit.About = strings.TrimSuffix(strings.TrimPrefix(a, "[["), "]]")
			res.Tasks[hit.About] = true
		}
		res.Hits = append(res.Hits, hit)
		return nil
	})

	// Notes carrying the origin (findings materialized from tainted events,
	// or authored directly with --origin). A note reaches its whole project's
	// briefs, so the project — not just the task — is tainted.
	projects, _ := ListProjects(w)
	for _, p := range projects {
		for _, kind := range []model.NoteKind{model.NoteFinding, model.NoteDecision, model.NoteRef} {
			docs, _ := ListNotes(w, p.Slug, kind)
			for _, doc := range docs {
				origin, _ := doc.Front.Get("origin")
				if origin == "" || !strings.Contains(origin, source) {
					continue
				}
				hit := TaintHit{Kind: "note", Origin: origin, Project: p.Slug}
				hit.ID, _ = doc.Front.Get("id")
				hit.Actor, _ = doc.Front.Get("created_by")
				if a, ok := doc.Front.Get("about"); ok {
					hit.About = strings.TrimSuffix(strings.TrimPrefix(a, "[["), "]]")
					res.Tasks[hit.About] = true
				}
				res.Hits = append(res.Hits, hit)
				if kind == model.NoteFinding {
					res.Projects[p.Slug] = true // findings surface project-wide
				}
			}
		}
	}
	return res, nil
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
	for proj := range r.Projects {
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
