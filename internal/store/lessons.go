package store

import (
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Lesson is a workspace-scoped note surfaced across project boundaries —
// PROPOSALS P1, the compounding loop. A finding in project A dies with A
// unless somebody marks it `scope: workspace`; then it reaches every brief
// in the workspace, which is the entire point.
type Lesson struct {
	Project string
	ID      string
	Actor   string
	Title   string
	Body    string
	Origin  string // carried forward so a promoted skill inherits it (SKILLS.md § 6)
}

// lessonKinds are the note kinds WorkspaceLessons surfaces cross-project as a
// "Lesson from other projects". Metric notes are deliberately excluded, so a
// scope: workspace metric reaches NO other project's brief. Taint reads the
// same set (SurfacesAsLesson) so its blast radius agrees with what actually
// crosses project boundaries instead of over-reporting a metric as tree-wide.
var lessonKinds = []model.NoteKind{model.NoteDecision, model.NoteFinding, model.NoteRef}

// SurfacesAsLesson reports whether a scope: workspace note of this kind would
// reach every project's brief via WorkspaceLessons.
func SurfacesAsLesson(kind model.NoteKind) bool {
	for _, k := range lessonKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// WorkspaceLessons collects scope: workspace notes from every project EXCEPT
// the excluded one — the current project's notes already reach its briefs
// through the findings and constraints sections; this is strictly the
// cross-project channel.
//
// Ranking is deliberately crude: all of them, in project/kind order, capped
// by the caller. Graph-proximity ranking (PROPOSALS P5) can refine this once
// there is evidence the crude version misranks in practice.
func WorkspaceLessons(w *workspace.Workspace, excludeProject string) []Lesson {
	projects, _ := ListProjects(w)
	var out []Lesson
	for _, p := range projects {
		if p.Slug == excludeProject {
			continue
		}
		for _, kind := range lessonKinds {
			notes, _ := ListNotes(w, p.Slug, kind)
			for _, n := range notes {
				if scope, _ := n.Front.Get("scope"); scope != "workspace" {
					continue
				}
				out = append(out, lessonFromDoc(p.Slug, n))
			}
		}
	}
	return out
}

// lessonFromDoc converts a scope:workspace note into the Lesson shape shared
// by WorkspaceLessons (brief assembly) and FindLesson (promotion).
func lessonFromDoc(project string, n *mdstore.Doc) Lesson {
	l := Lesson{Project: project}
	l.ID, _ = n.Front.Get("id")
	l.Actor, _ = n.Front.Get("created_by")
	l.Origin, _ = n.Front.Get("origin")
	var body strings.Builder
	for _, s := range n.Sections {
		if s.Level == 1 {
			l.Title = s.Title
			continue
		}
		if s.Title != "" {
			body.WriteString(s.Title + ": ")
		}
		body.WriteString(strings.TrimSpace(s.Content) + " ")
	}
	// Level-0/H1-nested content (the common shape after reparse).
	if body.Len() == 0 {
		for _, s := range n.Sections {
			body.WriteString(strings.TrimSpace(s.Content) + " ")
		}
	}
	l.Body = strings.TrimSpace(body.String())
	return l
}

// FindLesson resolves ref (a note id or its level-1 title) to a
// scope:workspace lesson note anywhere in the workspace. Unlike
// WorkspaceLessons — which is the cross-project brief channel and therefore
// excludes the reader's own project — promotion is workspace-wide and has no
// such exclusion: a lesson can be promoted from the caller's own project too.
func FindLesson(w *workspace.Workspace, ref string) (Lesson, error) {
	projects, _ := ListProjects(w)
	for _, p := range projects {
		for _, kind := range lessonKinds {
			notes, _ := ListNotes(w, p.Slug, kind)
			for _, n := range notes {
				if scope, _ := n.Front.Get("scope"); scope != "workspace" {
					continue
				}
				id, _ := n.Front.Get("id")
				l := lessonFromDoc(p.Slug, n)
				if id == ref || l.Title == ref {
					return l, nil
				}
			}
		}
	}
	return Lesson{}, ErrNotFound{Ref: ref}
}
