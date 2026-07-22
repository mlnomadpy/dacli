package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Risk is a parsed entry in the impact×likelihood matrix.
type Risk struct {
	Slug       string
	Project    string
	Title      string
	Impact     model.Level
	Likelihood model.Level
	Indicators []string
	Action     string
	Path       string
}

// Rank per the matrix: 1 mitigate now, 2 plan, 3 monitor only.
func (r *Risk) Rank() int {
	return model.Risk{Impact: r.Impact, Likelihood: r.Likelihood}.Rank()
}

// CreateRisk writes projects/<slug>/risks/<slug>.md.
func CreateRisk(w *workspace.Workspace, actor, project, title string, impact, likelihood model.Level, indicators []string, action string) (*Risk, error) {
	if _, err := LoadProject(w, project); err != nil {
		return nil, err
	}
	slug := Slugify(title)

	d := &mdstore.Doc{}
	d.Front.Set("id", "r-"+slug)
	d.Front.Set("kind", string(model.KindRisk))
	d.Front.Set("created", now())
	d.Front.Set("created_by", actor)
	d.Front.Set("impact", string(impact))
	d.Front.Set("likelihood", string(likelihood))

	d.Sections = []mdstore.Section{{Level: 1, Title: title, Content: ""}}
	var ind strings.Builder
	for _, i := range indicators {
		ind.WriteString("- " + i + "\n")
	}
	d.Sections = append(d.Sections,
		mdstore.Section{Level: 2, Title: "Indicators", Content: ind.String()},
		mdstore.Section{Level: 2, Title: "Action", Content: action + "\n"},
	)

	path := filepath.Join(w.RisksDir(project), slug+".md")
	if err := mdstore.WriteFile(path, d); err != nil {
		return nil, err
	}
	return &Risk{Slug: slug, Project: project, Title: title, Impact: impact, Likelihood: likelihood, Indicators: indicators, Action: action, Path: path}, nil
}

// ListRisks returns a project's risks, rank-1 first.
func ListRisks(w *workspace.Workspace, project string) ([]*Risk, error) {
	dir := w.RisksDir(project)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no risks dir yet is not an error
		}
		return nil, err // a real I/O/permission failure must not read as "empty"
	}
	var out []*Risk
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		d, err := mdstore.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		r := &Risk{Slug: strings.TrimSuffix(e.Name(), ".md"), Project: project, Path: filepath.Join(dir, e.Name())}
		if v, ok := d.Front.Get("impact"); ok {
			r.Impact = model.Level(v)
		}
		if v, ok := d.Front.Get("likelihood"); ok {
			r.Likelihood = model.Level(v)
		}
		for _, s := range d.Sections {
			switch {
			case s.Level == 1:
				r.Title = s.Title
				// The body under the H1 may hold indicators/action if the file
				// was hand-written without subsections; subsections win below.
			case strings.EqualFold(s.Title, "Indicators"):
				r.Indicators = mdstore.Bullets(s.Content)
			case strings.EqualFold(s.Title, "Action"):
				r.Action = strings.TrimSpace(s.Content)
			}
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rank() != out[j].Rank() {
			return out[i].Rank() < out[j].Rank()
		}
		return out[i].Slug < out[j].Slug
	})
	return out, nil
}

// --- Glossary ---

// GlossaryAdd appends one term. The glossary is the counter to the
// vague-noun ambiguity category: one definition every agent shares.
func GlossaryAdd(w *workspace.Workspace, actor, project, term, def string) error {
	if _, err := LoadProject(w, project); err != nil {
		return err
	}
	path := w.GlossaryPath(project)
	d, err := mdstore.ReadFile(path)
	if os.IsNotExist(err) {
		d = &mdstore.Doc{}
		d.Front.Set("id", "g-"+project)
		d.Front.Set("kind", string(model.KindNote))
		d.Front.Set("note_kind", string(model.NoteRef))
		d.Front.Set("created", now())
		d.Front.Set("created_by", actor)
		d.Sections = []mdstore.Section{{Level: 1, Title: "Glossary", Content: ""}}
	} else if err != nil {
		return err
	}
	s, _ := d.Section("Glossary")
	d.SetSection("Glossary", s.Content+fmt.Sprintf("- **%s** — %s\n", term, def))
	return mdstore.WriteFile(path, d)
}

// GlossaryRead returns the rendered glossary body, or "".
func GlossaryRead(w *workspace.Workspace, project string) string {
	d, err := mdstore.ReadFile(w.GlossaryPath(project))
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, s := range d.Sections {
		b.WriteString(s.Content)
	}
	return b.String()
}
