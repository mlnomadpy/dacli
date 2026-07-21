// Package store is L2: object CRUD over mdstore, enforcing ownership.
//
// Status is folder position, ids are ULIDs, and the NNN filename prefix is a
// display alias assigned by the single allocator (the creating owner), per
// docs/FORMAT.md. Nothing here touches the event log — that is L3's job.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// ErrNotFound distinguishes "no such object" from operational failures, so
// the CLI can map it to exit code 4.
type ErrNotFound struct{ Ref string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("not found: %s", e.Ref) }

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// Slugify turns a title into a filename-safe slug.
func Slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// --- Projects ---

// Project is the parsed view store works with; heavier typing waits for the
// brief assembler, which is the real consumer.
type Project struct {
	Slug  string
	Doc   *mdstore.Doc
	Path  string
	Title string
	Stage string
}

// CreateProject writes projects/<slug>/project.md with the structural
// sections the brief assembler reads by heading.
func CreateProject(w *workspace.Workspace, actor, title, slug, goal, stage string) (*Project, error) {
	if slug == "" {
		slug = Slugify(title)
	}
	path := w.ProjectPath(slug)
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("project %q already exists", slug)
	}
	if stage == "" {
		// New projects start at the widest point of the Cone: claiming more
		// certainty than "we just defined this" would be a lie in a field.
		stage = "definition"
	}

	d := &mdstore.Doc{}
	d.Front.Set("id", "p-"+slug)
	d.Front.Set("kind", string(model.KindProject))
	d.Front.Set("created", now())
	d.Front.Set("created_by", actor)
	d.Front.Set("status", "active")
	d.Front.Set("stage", stage)
	d.Sections = []mdstore.Section{
		{Level: 1, Title: title, Content: ""},
		{Level: 2, Title: "Goal", Content: goal + "\n"},
		{Level: 2, Title: "Constraints", Content: ""},
		{Level: 2, Title: "Out of scope", Content: ""},
		{Level: 2, Title: "Success criteria", Content: ""},
	}
	if err := mdstore.WriteFile(path, d); err != nil {
		return nil, err
	}
	return &Project{Slug: slug, Doc: d, Path: path, Title: title, Stage: stage}, nil
}

func LoadProject(w *workspace.Workspace, slug string) (*Project, error) {
	path := w.ProjectPath(slug)
	d, err := mdstore.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound{Ref: "project/" + slug}
		}
		return nil, err
	}
	p := &Project{Slug: slug, Doc: d, Path: path}
	p.Stage, _ = d.Front.Get("stage")
	for _, s := range d.Sections {
		if s.Level == 1 {
			p.Title = s.Title
			break
		}
	}
	return p, nil
}

func ListProjects(w *workspace.Workspace) ([]*Project, error) {
	entries, err := os.ReadDir(w.ProjectsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*Project
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), "_") {
			continue
		}
		p, err := LoadProject(w, e.Name())
		if err != nil {
			continue // a broken project dir shouldn't hide the others
		}
		out = append(out, p)
	}
	return out, nil
}

// --- Tasks ---

type Task struct {
	ID      string
	Seq     int
	Slug    string
	Project string
	Status  model.Status
	Title   string
	Doc     *mdstore.Doc
	Path    string
}

func (t *Task) Owner() string    { v, _ := t.Doc.Front.Get("owner"); return v }
func (t *Task) Priority() string { v, _ := t.Doc.Front.Get("priority"); return v }

// Acceptance returns the task's acceptance checkboxes.
func (t *Task) Acceptance() []mdstore.Checkbox {
	s, ok := t.Doc.Section("Acceptance")
	if !ok {
		return nil
	}
	return mdstore.Checkboxes(s.Content)
}

// TaskOpts carries creation options; zero values are simply omitted.
type TaskOpts struct {
	Priority  string
	Estimate  string // "o,m,p"
	Accept    []string
	SoThat    string
	Context   string
	DependsOn []string // "ref" or "ref:SS" etc.
}

// Dep is one typed dependency. SS is what makes two tasks genuinely
// parallel-safe; everything else blocks.
type Dep struct {
	Ref  string
	Type string // FS | SS | FF | SF; FS when unspecified
}

// Deps parses the task's depends_on list.
func (t *Task) Deps() []Dep {
	var out []Dep
	for _, raw := range t.Doc.Front.GetList("depends_on") {
		d := Dep{Ref: raw, Type: "FS"}
		if i := strings.Index(raw, ":"); i > 0 {
			d.Ref, d.Type = raw[:i], strings.ToUpper(raw[i+1:])
		}
		out = append(out, d)
	}
	return out
}

// Estimate returns the three-point estimate if present and valid.
func (t *Task) Estimate() (spm.ThreePoint, bool) {
	m := t.Doc.Front.GetMap("estimate")
	if m == nil {
		return spm.ThreePoint{}, false
	}
	var tp spm.ThreePoint
	fmt.Sscanf(m["optimistic"], "%g", &tp.Optimistic)
	fmt.Sscanf(m["probable"], "%g", &tp.Probable)
	fmt.Sscanf(m["pessimistic"], "%g", &tp.Pessimistic)
	return tp, tp.Valid() == nil && tp.Pessimistic > 0
}

// CreateTask allocates the next NNN in the project (we are the owner at
// creation, so we are the single allocator the format requires) and writes
// the task into tasks/open/.
func CreateTask(w *workspace.Workspace, actor, project, title string, opts TaskOpts) (*Task, error) {
	if _, err := LoadProject(w, project); err != nil {
		return nil, err
	}
	seq := 1
	all, _ := ListTasks(w, project, "")
	for _, t := range all {
		if t.Seq >= seq {
			seq = t.Seq + 1
		}
	}

	id := "t-" + ulid.New()
	slug := Slugify(title)

	d := &mdstore.Doc{}
	d.Front.Set("id", id)
	d.Front.Set("kind", string(model.KindTask))
	d.Front.Set("created", now())
	d.Front.Set("created_by", actor)
	d.Front.Set("owner", actor)
	if opts.Priority != "" {
		d.Front.Set("priority", opts.Priority)
	}
	if opts.Estimate != "" {
		parts := strings.Split(opts.Estimate, ",")
		if len(parts) != 3 {
			return nil, fmt.Errorf("estimate must be three-point o,m,p — a scalar hides the risk (got %q)", opts.Estimate)
		}
		d.Front.Set("estimate", fmt.Sprintf("{optimistic: %s, probable: %s, pessimistic: %s}",
			strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])))
	}
	if len(opts.DependsOn) > 0 {
		d.Front.Set("depends_on", "["+strings.Join(opts.DependsOn, ", ")+"]")
	}

	d.Sections = []mdstore.Section{{Level: 1, Title: title, Content: ""}}
	if opts.SoThat != "" {
		d.Sections = append(d.Sections, mdstore.Section{Level: 2, Title: "So that", Content: opts.SoThat + "\n"})
	}
	if opts.Context != "" {
		d.Sections = append(d.Sections, mdstore.Section{Level: 2, Title: "Context", Content: opts.Context + "\n"})
	}
	var boxes []mdstore.Checkbox
	for _, a := range opts.Accept {
		boxes = append(boxes, mdstore.Checkbox{Text: a})
	}
	d.Sections = append(d.Sections,
		mdstore.Section{Level: 2, Title: "Acceptance", Content: mdstore.RenderCheckboxes(boxes)},
		mdstore.Section{Level: 2, Title: "Log", Content: ""},
	)

	path := filepath.Join(w.TasksDir(project, model.StatusOpen), fmt.Sprintf("%03d-%s.md", seq, slug))
	if err := mdstore.WriteFile(path, d); err != nil {
		return nil, err
	}
	return &Task{ID: id, Seq: seq, Slug: slug, Project: project, Status: model.StatusOpen, Title: title, Doc: d, Path: path}, nil
}

// ListTasks returns tasks for a project (or all projects if project == ""),
// optionally filtered by status. Status comes from the folder, never from
// frontmatter.
func ListTasks(w *workspace.Workspace, project string, status model.Status) ([]*Task, error) {
	var projects []string
	if project != "" {
		projects = []string{project}
	} else {
		ps, err := ListProjects(w)
		if err != nil {
			return nil, err
		}
		for _, p := range ps {
			projects = append(projects, p.Slug)
		}
	}

	var out []*Task
	for _, proj := range projects {
		for _, st := range model.AllStatuses {
			if status != "" && st != status {
				continue
			}
			dir := w.TasksDir(proj, st)
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				t, err := loadTaskFile(filepath.Join(dir, e.Name()), proj, st)
				if err != nil {
					continue
				}
				out = append(out, t)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Project != out[j].Project {
			return out[i].Project < out[j].Project
		}
		return out[i].Seq < out[j].Seq
	})
	return out, nil
}

func loadTaskFile(path, project string, st model.Status) (*Task, error) {
	d, err := mdstore.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t := &Task{Project: project, Status: st, Doc: d, Path: path}
	t.ID, _ = d.Front.Get("id")
	base := strings.TrimSuffix(filepath.Base(path), ".md")
	if i := strings.Index(base, "-"); i > 0 {
		t.Seq, _ = strconv.Atoi(base[:i])
		t.Slug = base[i+1:]
	}
	for _, s := range d.Sections {
		if s.Level == 1 {
			t.Title = s.Title
			break
		}
	}
	return t, nil
}

// FindTask resolves a ref — ULID id, t-<ULID>, NNN, or slug — searching all
// projects. Ambiguity is an error, not a guess: acting on the wrong task
// because a slug matched twice is the silent version of a collision.
func FindTask(w *workspace.Workspace, ref string) (*Task, error) {
	all, err := ListTasks(w, "", "")
	if err != nil {
		return nil, err
	}
	var hits []*Task
	for _, t := range all {
		if t.ID == ref || strings.TrimPrefix(t.ID, "t-") == ref || t.Slug == ref ||
			fmt.Sprintf("%03d", t.Seq) == ref || strconv.Itoa(t.Seq) == ref ||
			fmt.Sprintf("%03d-%s", t.Seq, t.Slug) == ref {
			hits = append(hits, t)
		}
	}
	switch len(hits) {
	case 0:
		return nil, ErrNotFound{Ref: ref}
	case 1:
		return hits[0], nil
	default:
		var names []string
		for _, h := range hits {
			names = append(names, fmt.Sprintf("%s/%03d-%s", h.Project, h.Seq, h.Slug))
		}
		return nil, fmt.Errorf("ref %q is ambiguous: %s", ref, strings.Join(names, ", "))
	}
}

// SaveTask rewrites a task in place.
func SaveTask(t *Task) error { return mdstore.WriteFile(t.Path, t.Doc) }

// MoveTask changes status by moving the file — the folder is the single
// source of truth, so this is the only way status ever changes.
func MoveTask(w *workspace.Workspace, t *Task, to model.Status) error {
	dst := filepath.Join(w.TasksDir(t.Project, to), filepath.Base(t.Path))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(t.Path, dst); err != nil {
		return err
	}
	t.Path, t.Status = dst, to
	return nil
}

// AppendLog adds a timestamped line to the task's ## Log.
func AppendLog(t *Task, line string) {
	s, _ := t.Doc.Section("Log")
	t.Doc.SetSection("Log", s.Content+fmt.Sprintf("- %s %s\n", now(), line))
}

// --- Notes ---

type NoteOpts struct {
	About    string
	Rejected string
	Because  string
	Severity string
	Scope    string // project | workspace — the P1 capture field
	Body     string
}

// CreateNote writes a decision, finding, metric, or ref note.
func CreateNote(w *workspace.Workspace, actor, project string, kind model.NoteKind, title string, opts NoteOpts) (string, error) {
	if _, err := LoadProject(w, project); err != nil {
		return "", err
	}
	if kind == model.NoteDecision && opts.Rejected == "" {
		// A decision without a rejection cannot be safely revisited; refusing
		// here is cheaper than dacli lint flagging it later.
		return "", fmt.Errorf("a decision must record what was rejected (--rejected)")
	}

	slug := Slugify(title)
	prefix := map[model.NoteKind]string{
		model.NoteDecision: "d-", model.NoteFinding: "f-", model.NoteMetric: "m-", model.NoteRef: "r-",
	}[kind]

	d := &mdstore.Doc{}
	d.Front.Set("id", prefix+slug)
	d.Front.Set("kind", string(model.KindNote))
	d.Front.Set("note_kind", string(kind))
	d.Front.Set("created", now())
	d.Front.Set("created_by", actor)
	if opts.About != "" {
		d.Front.Set("about", "[["+opts.About+"]]")
	}
	if opts.Severity != "" {
		d.Front.Set("severity", opts.Severity)
	}
	if opts.Scope != "" {
		d.Front.Set("scope", opts.Scope)
	}

	d.Sections = []mdstore.Section{{Level: 1, Title: title, Content: ""}}
	if kind == model.NoteDecision {
		d.Sections = append(d.Sections,
			mdstore.Section{Level: 2, Title: "Chose", Content: firstNonEmpty(opts.Body, title) + "\n"},
			mdstore.Section{Level: 2, Title: "Rejected", Content: opts.Rejected + "\n"},
			mdstore.Section{Level: 2, Title: "Because", Content: opts.Because + "\n"},
		)
	} else if opts.Body != "" {
		d.Sections = append(d.Sections, mdstore.Section{Level: 0, Content: opts.Body + "\n"})
	}

	path := filepath.Join(w.NotesDir(project, kind), slug+".md")
	// Same-titled notes must not clobber each other — sync materializes
	// findings from events, and two agents finding the same thing is normal.
	if _, err := os.Stat(path); err == nil {
		suffix := strings.ToLower(ulid.New())
		path = filepath.Join(w.NotesDir(project, kind), slug+"-"+suffix[len(suffix)-6:]+".md")
		d.Front.Set("id", prefix+slug+"-"+suffix[len(suffix)-6:])
	}
	if err := mdstore.WriteFile(path, d); err != nil {
		return "", err
	}
	return path, nil
}

// ListNotes returns parsed notes of one kind for a project.
func ListNotes(w *workspace.Workspace, project string, kind model.NoteKind) ([]*mdstore.Doc, error) {
	dir := w.NotesDir(project, kind)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	var out []*mdstore.Doc
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if d, err := mdstore.ReadFile(filepath.Join(dir, e.Name())); err == nil {
			out = append(out, d)
		}
	}
	return out, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
