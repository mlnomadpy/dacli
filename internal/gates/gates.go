// Package gates implements project templates and stage gates: what kind of
// project this is, which stages it passes through, and what must be TRUE —
// not merely present — to leave each stage.
//
// The predicate vocabulary is deliberately small. Most predicates are
// non-scriptable, and the one that carries the most weight is FILLED, not
// present — an agent under budget pressure will produce a heading with "TBD"
// under it, and "make things better and handle the edge cases properly" is
// exactly as empty as "TBD".
//
// Two predicates DO reach outside the markdown: `artifact` (a path exists) and
// `command` (a shell command exits 0). This revises an earlier stated rule that
// no predicate may run code. The original reasoning — a scriptable gate becomes
// a place people hide logic, and stops being auditable — is sound but was
// answered by the wrong mechanism. Every other predicate inspects documents, so
// a stage could be "passed" with a broken build and a failing test suite: the
// gates certified the paperwork, not the software (dacli 186). Auditability is
// preserved differently: the command is written verbatim in the template
// manifest, so it is reviewed with the template rather than hidden in code, and
// its exit status is reported in `stage status` like any other check. A gate
// that cannot say "the tests pass" is not a quality gate.
package gates

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

//go:embed tpl
var embedded embed.FS

// Predicate is one gate check, parsed from a manifest bullet.
type Predicate struct {
	Kind string // project_sections | glossary | decisions | tasks | risks | retro | artifact | command
	Arg  string
}

// Stage is one controlled step.
type Stage struct {
	Name       string
	Cone       string   // which Cone-of-Uncertainty stage this maps to
	Phase      string   // the lifecycle phase: discovery|research|planning|design|implementation|review|release
	Allow      []string // role KINDS permitted to act in this phase; empty = any
	Predicates []Predicate
}

// AllowsKind reports whether a role of the given kind may act in this phase.
// An empty Allow list is permissive; a role with no kind always passes
// (phase gating is opt-in per role).
func (s Stage) AllowsKind(kind string) bool {
	if kind == "" || len(s.Allow) == 0 {
		return true
	}
	for _, a := range s.Allow {
		if a == kind {
			return true
		}
	}
	return false
}

// Template is a parsed manifest.
type Template struct {
	Name     string
	Summary  string
	Cost     string
	Stages   []Stage
	Origin   string // embedded | workspace
	Manifest string // raw, for `template show`
}

// Load returns every template: workspace vendored files win over embedded
// defaults of the same name — the nearest-wins rule everything else uses.
func Load(w *workspace.Workspace) ([]Template, error) {
	byName := map[string]Template{}
	var order []string

	entries, _ := embedded.ReadDir("tpl")
	for _, e := range entries {
		raw, _ := embedded.ReadFile("tpl/" + e.Name())
		if t, err := parse(string(raw), "embedded"); err == nil {
			byName[t.Name] = t
			order = append(order, t.Name)
		}
	}
	if w != nil {
		files, _ := os.ReadDir(w.TemplatesDir())
		for _, e := range files {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(w.TemplatesDir(), e.Name()))
			if err != nil {
				continue
			}
			if t, err := parse(string(raw), "workspace"); err == nil {
				if _, known := byName[t.Name]; !known {
					order = append(order, t.Name)
				}
				byName[t.Name] = t
			}
		}
	}

	out := make([]Template, 0, len(order))
	for _, n := range order {
		out = append(out, byName[n])
	}
	return out, nil
}

// Get finds one template by name.
func Get(w *workspace.Workspace, name string) (Template, error) {
	ts, err := Load(w)
	if err != nil {
		return Template{}, err
	}
	for _, t := range ts {
		if t.Name == name {
			return t, nil
		}
	}
	return Template{}, store.ErrNotFound{Ref: "template " + name}
}

// Vendor copies an embedded template into the workspace for editing.
func Vendor(w *workspace.Workspace, name string) (string, error) {
	raw, err := embedded.ReadFile("tpl/" + name + ".md")
	if err != nil {
		return "", store.ErrNotFound{Ref: "embedded template " + name}
	}
	path := filepath.Join(w.TemplatesDir(), name+".md")
	if err := os.MkdirAll(w.TemplatesDir(), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("template %q is already vendored at %s", name, path)
	}
	return path, os.WriteFile(path, raw, 0o644)
}

// parse reads a manifest: frontmatter (name/summary/cost) + one section per
// "## stage: <name>" carrying a cone: line and predicate bullets. Sectioned
// markdown rather than nested YAML, because the flat frontmatter dialect is
// a deliberate boundary (v1 deviation, recorded in TEMPLATES.md).
func parse(raw, origin string) (Template, error) {
	d, err := mdstore.Parse(raw)
	if err != nil {
		return Template{}, err
	}
	t := Template{Origin: origin, Manifest: raw}
	t.Name, _ = d.Front.Get("name")
	t.Summary, _ = d.Front.Get("summary")
	t.Cost, _ = d.Front.Get("cost")
	if t.Name == "" {
		return Template{}, fmt.Errorf("template manifest has no name")
	}
	for _, s := range d.Sections {
		if !strings.HasPrefix(strings.ToLower(s.Title), "stage:") {
			continue
		}
		st := Stage{Name: strings.TrimSpace(s.Title[len("stage:"):])}
		for _, line := range strings.Split(s.Content, "\n") {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "cone:"):
				st.Cone = strings.TrimSpace(line[len("cone:"):])
			case strings.HasPrefix(line, "phase:"):
				st.Phase = strings.TrimSpace(line[len("phase:"):])
			case strings.HasPrefix(line, "allow:"):
				for _, k := range strings.Split(line[len("allow:"):], ",") {
					if k = strings.TrimSpace(k); k != "" {
						st.Allow = append(st.Allow, k)
					}
				}
			}
		}
		for _, b := range mdstore.Bullets(s.Content) {
			kind, arg, _ := strings.Cut(b, ":")
			st.Predicates = append(st.Predicates, Predicate{
				Kind: strings.TrimSpace(kind), Arg: strings.TrimSpace(arg),
			})
		}
		t.Stages = append(t.Stages, st)
	}
	return t, nil
}

// Attach binds a template to a project: template name, first stage, and the
// stage's cone written into project.md.
func Attach(w *workspace.Workspace, projectSlug, tmplName string) (Stage, error) {
	t, err := Get(w, tmplName)
	if err != nil {
		return Stage{}, err
	}
	p, err := store.LoadProject(w, projectSlug)
	if err != nil {
		return Stage{}, err
	}
	p.Doc.Front.Set("template", t.Name)
	if len(t.Stages) == 0 {
		// solo: a template with no stages attaches as already-complete.
		p.Doc.Front.Set("template_stage", "complete")
		return Stage{Name: "complete"}, mdstore.WriteFile(p.Path, p.Doc)
	}
	first := t.Stages[0]
	p.Doc.Front.Set("template_stage", first.Name)
	if first.Cone != "" {
		p.Doc.Front.Set("stage", first.Cone)
	}
	writePhase(p, first) // so briefs and spawn read the phase without loading the template
	return first, mdstore.WriteFile(p.Path, p.Doc)
}

// Check is one evaluated predicate.
type Check struct {
	Desc string
	OK   bool
	Why  string // when !OK: present-vs-filled distinctions live here
}

// Status evaluates the project's current stage.
type ProjectStatus struct {
	Template string
	Stage    string
	Checks   []Check
	Complete bool // template_stage == complete, or no template (solo)
	Next     *Stage
}

// Status evaluates where a project stands against its template.
func Status(w *workspace.Workspace, projectSlug string) (*ProjectStatus, error) {
	st, _, err := status(w, projectSlug)
	return st, err
}

// status is Status plus the loaded project, so callers that also mutate the
// project (Advance) reuse the single load instead of reading it off disk again.
func status(w *workspace.Workspace, projectSlug string) (*ProjectStatus, *store.Project, error) {
	p, err := store.LoadProject(w, projectSlug)
	if err != nil {
		return nil, nil, err
	}
	st := &ProjectStatus{}
	st.Template, _ = p.Doc.Front.Get("template")
	st.Stage, _ = p.Doc.Front.Get("template_stage")
	if st.Template == "" || st.Template == "solo" || st.Stage == "complete" {
		st.Complete = true
		return st, p, nil
	}
	t, err := Get(w, st.Template)
	if err != nil {
		return nil, nil, err
	}
	for i, s := range t.Stages {
		if s.Name != st.Stage {
			continue
		}
		for _, pred := range s.Predicates {
			st.Checks = append(st.Checks, evaluate(w, p, pred))
		}
		if i+1 < len(t.Stages) {
			next := t.Stages[i+1]
			st.Next = &next
		}
		return st, p, nil
	}
	return nil, nil, fmt.Errorf("project is at stage %q, which template %s does not define — the manifest changed under it", st.Stage, st.Template)
}

// Advance moves to the next stage if every check passes; the caller turns
// unmet checks into the exit-3 refusal.
func Advance(w *workspace.Workspace, projectSlug string) (newStage string, unmet []Check, err error) {
	st, p, err := status(w, projectSlug)
	if err != nil {
		return "", nil, err
	}
	if st.Complete {
		return "complete", nil, nil
	}
	for _, c := range st.Checks {
		if !c.OK {
			unmet = append(unmet, c)
		}
	}
	if len(unmet) > 0 {
		return "", unmet, nil
	}
	// p is the project Status already loaded — reuse it rather than reading
	// the same file off disk a second time.
	if st.Next == nil {
		p.Doc.Front.Set("template_stage", "complete")
		return "complete", nil, mdstore.WriteFile(p.Path, p.Doc)
	}
	p.Doc.Front.Set("template_stage", st.Next.Name)
	if st.Next.Cone != "" {
		// Passing a gate narrows the Cone: every estimate in the project now
		// reports tighter, which is the honest version of "we know more now".
		p.Doc.Front.Set("stage", st.Next.Cone)
	}
	writePhase(p, *st.Next)
	return st.Next.Name, nil, mdstore.WriteFile(p.Path, p.Doc)
}

// writePhase records the current phase and its allowed role-kinds onto the
// project, so the brief assembler and spawn gate read them cheaply — no
// template load on the hot path.
func writePhase(p *store.Project, s Stage) {
	if s.Phase == "" {
		p.Doc.Front.Delete("phase")
		p.Doc.Front.Delete("phase_allows")
		return
	}
	p.Doc.Front.Set("phase", s.Phase)
	if len(s.Allow) > 0 {
		p.Doc.Front.SetList("phase_allows", s.Allow)
	} else {
		p.Doc.Front.Delete("phase_allows")
	}
}

// Phase is a project's current lifecycle position — what kind of work is
// appropriate now, and which role-kinds may act.
type Phase struct {
	Name   string   // discovery | planning | implementation | ...
	Allows []string // role kinds permitted; empty = any
	Gated  bool     // false for solo / untemplated projects
}

// PhaseFor returns a project's current phase, read from its frontmatter (set
// at attach/advance). Cheap: no template parse.
func PhaseFor(w *workspace.Workspace, projectSlug string) (Phase, error) {
	p, err := store.LoadProject(w, projectSlug)
	if err != nil {
		return Phase{}, err
	}
	name, ok := p.Doc.Front.Get("phase")
	if !ok || name == "" {
		return Phase{Gated: false}, nil
	}
	return Phase{Name: name, Allows: p.Doc.Front.GetList("phase_allows"), Gated: true}, nil
}

// AllowsKind reports whether a role of this kind may act in the phase.
func (ph Phase) AllowsKind(kind string) bool {
	if !ph.Gated || kind == "" || len(ph.Allows) == 0 {
		return true
	}
	for _, a := range ph.Allows {
		if a == kind {
			return true
		}
	}
	return false
}

func evaluate(w *workspace.Workspace, p *store.Project, pred Predicate) Check {
	switch pred.Kind {
	case "project_sections":
		var missing []string
		for _, name := range strings.Split(pred.Arg, "|") {
			name = strings.TrimSpace(name)
			s, ok := p.Doc.Section(name)
			if !ok {
				missing = append(missing, name+" (missing)")
				continue
			}
			if why := unfilled(s.Content); why != "" {
				missing = append(missing, fmt.Sprintf("%s (present but not filled: %s)", name, why))
			}
		}
		return Check{Desc: "project sections filled: " + pred.Arg, OK: len(missing) == 0, Why: strings.Join(missing, "; ")}

	case "glossary":
		n := 0
		for _, line := range strings.Split(store.GlossaryRead(w, p.Slug), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "- ") {
				n++
			}
		}
		want := 1
		fmt.Sscanf(pred.Arg, "min_terms %d", &want)
		return Check{Desc: fmt.Sprintf("glossary has ≥%d terms", want), OK: n >= want,
			Why: fmt.Sprintf("%d defined", n)}

	case "decisions":
		notes, _ := store.ListNotes(w, p.Slug, "decision")
		want := 1
		fmt.Sscanf(pred.Arg, "min %d", &want)
		// The gate's own Desc promises "with a rejection", and the rejected
		// alternative is the valuable part of a decision note — so verify each
		// note actually carries a non-empty Rejected section, not merely that N
		// decision notes exist. A rejection-free decision no longer clears the
		// design gate. (A real rejection can be terse — "async queue" — so this
		// is a non-empty check, not the full ≥20-char `unfilled` bar.)
		n := 0
		for _, d := range notes {
			if s, ok := d.Section("Rejected"); ok && strings.TrimSpace(s.Content) != "" {
				n++
			}
		}
		return Check{Desc: fmt.Sprintf("≥%d decision(s) with a rejection", want), OK: n >= want,
			Why: fmt.Sprintf("%d of %d recorded carry a rejection", n, len(notes))}

	case "tasks":
		tasks, _ := store.ListTasks(w, p.Slug, "")
		switch pred.Arg {
		case "all_have_acceptance":
			var bad []string
			for _, t := range tasks {
				if len(t.Acceptance()) == 0 {
					bad = append(bad, fmt.Sprintf("%03d", t.Seq))
				}
			}
			return Check{Desc: "every task has acceptance criteria", OK: len(bad) == 0, Why: strings.Join(bad, ", ")}
		case "all_have_estimate":
			var bad []string
			for _, t := range tasks {
				if _, ok := t.Estimate(); !ok {
					bad = append(bad, fmt.Sprintf("%03d", t.Seq))
				}
			}
			return Check{Desc: "every task has a three-point estimate", OK: len(bad) == 0, Why: strings.Join(bad, ", ")}
		case "musts_done":
			var open []string
			for _, t := range tasks {
				if t.Priority() == "must" && t.Status != "done" {
					open = append(open, fmt.Sprintf("%03d", t.Seq))
				}
			}
			return Check{Desc: "every must task is done", OK: len(open) == 0, Why: strings.Join(open, ", ")}
		}

	case "risks":
		if pred.Arg == "rank1_have_action" {
			risks, _ := store.ListRisks(w, p.Slug)
			var bad []string
			for _, r := range risks {
				if r.Rank() == 1 && strings.TrimSpace(r.Action) == "" {
					bad = append(bad, r.Slug)
				}
			}
			return Check{Desc: "every rank-1 risk has an action plan", OK: len(bad) == 0, Why: strings.Join(bad, ", ")}
		}

	case "retro":
		notes, _ := store.ListNotes(w, p.Slug, "ref")
		for _, n := range notes {
			for _, s := range n.Sections {
				if s.Level == 1 && strings.HasPrefix(s.Title, "Retro:") {
					return Check{Desc: "a retro is recorded", OK: true}
				}
			}
		}
		return Check{Desc: "a retro is recorded", OK: false, Why: "none found"}

	case "artifact":
		// Existence of a real file or directory, relative to the workspace root.
		// No code runs; this is the auditable half of "the software exists".
		var missing []string
		for _, rel := range strings.Split(pred.Arg, "|") {
			rel = strings.TrimSpace(rel)
			if rel == "" {
				continue
			}
			if !workspace.SafeRelPath(rel) {
				missing = append(missing, rel+" (invalid path)")
				continue
			}
			if _, err := os.Stat(filepath.Join(w.Root, rel)); err != nil {
				missing = append(missing, rel)
			}
		}
		return Check{Desc: "artifact exists: " + pred.Arg, OK: len(missing) == 0, Why: strings.Join(missing, "; ")}

	case "command":
		// The only predicate that executes. Bounded by a deadline so a hung
		// build can never wedge a gate check, and its failure output is
		// surfaced verbatim so `stage status` explains WHY the gate is shut.
		if strings.TrimSpace(pred.Arg) == "" {
			return Check{Desc: "command", OK: false, Why: "no command given in the manifest"}
		}
		cctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()
		c := exec.CommandContext(cctx, "sh", "-c", pred.Arg)
		c.Dir = w.Root
		out, err := c.CombinedOutput()
		desc := fmt.Sprintf("`%s` exits 0", pred.Arg)
		if cctx.Err() == context.DeadlineExceeded {
			return Check{Desc: desc, OK: false, Why: fmt.Sprintf("timed out after %s", commandTimeout)}
		}
		if err != nil {
			return Check{Desc: desc, OK: false, Why: strings.TrimSpace(lastLines(string(out), 3))}
		}
		return Check{Desc: desc, OK: true}

	case "coverage":
		// `coverage: <min> <command>` — the first token is the floor, the rest is
		// the command. Test coverage is the one quality signal a reader of a
		// repository can check independently, so it deserves a gate rather than
		// living only in a CI log nobody reads (dacli 187).
		minStr, cmdStr, found := strings.Cut(strings.TrimSpace(pred.Arg), " ")
		if !found || strings.TrimSpace(cmdStr) == "" {
			return Check{Desc: "coverage", OK: false, Why: "usage: coverage: <min-percent> <command>"}
		}
		min, perr := strconv.ParseFloat(strings.TrimSuffix(minStr, "%"), 64)
		if perr != nil {
			return Check{Desc: "coverage", OK: false, Why: fmt.Sprintf("%q is not a percentage", minStr)}
		}
		cctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()
		c := exec.CommandContext(cctx, "sh", "-c", cmdStr)
		c.Dir = w.Root
		out, err := c.CombinedOutput()
		desc := fmt.Sprintf("coverage >= %g%%", min)
		if cctx.Err() == context.DeadlineExceeded {
			return Check{Desc: desc, OK: false, Why: fmt.Sprintf("timed out after %s", commandTimeout)}
		}
		if err != nil {
			return Check{Desc: desc, OK: false, Why: "the coverage command failed: " + lastLines(string(out), 2)}
		}
		got, ok := lastPercent(string(out))
		if !ok {
			return Check{Desc: desc, OK: false, Why: "no percentage found in the command's output"}
		}
		return Check{
			Desc: desc,
			OK:   got >= min,
			Why:  fmt.Sprintf("measured %.1f%%", got),
		}
	}
	return Check{Desc: pred.Kind + ": " + pred.Arg, OK: false, Why: "unknown predicate — the manifest and this build disagree"}
}

// commandTimeout bounds a `command:` predicate. A gate check runs inside
// interactive commands (`stage status`, `stage advance`), so an unbounded
// build would hang the CLI.
const commandTimeout = 10 * time.Minute

// lastPercent extracts the final percentage in s. Coverage tools all print one
// — `coverage: 87.3% of statements` (go), `TOTAL … 95%` (pytest-cov),
// `All files | 82.4 |` (jest) — and in each the summary figure is the last one
// emitted, so the command can be shaped to end with the number that matters
// (e.g. piping `go tool cover -func` through `tail -1`).
func lastPercent(s string) (float64, bool) {
	val, ok := 0.0, false
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		// Walk back over the number preceding the sign.
		j := i
		for j > 0 && (s[j-1] >= '0' && s[j-1] <= '9' || s[j-1] == '.') {
			j--
		}
		if j == i {
			continue
		}
		if f, err := strconv.ParseFloat(strings.TrimSuffix(s[j:i], "."), 64); err == nil {
			val, ok = f, true
		}
	}
	return val, ok
}

// lastLines returns the final n non-empty lines of s — enough to explain a
// failure without pasting an entire build log into the gate report.
func lastLines(s string, n int) string {
	var keep []string
	for _, ln := range strings.Split(s, "\n") {
		if strings.TrimSpace(ln) != "" {
			keep = append(keep, ln)
		}
	}
	if len(keep) > n {
		keep = keep[len(keep)-n:]
	}
	return strings.Join(keep, "; ")
}

// unfilled is the filled-not-present rule: empty, placeholder-bearing, too
// short, or majorly ambiguous content does not satisfy a gate. Returns the
// reason, or "" when the content is genuinely filled.
func unfilled(content string) string {
	c := strings.TrimSpace(content)
	if c == "" {
		return "empty"
	}
	for _, ph := range []string{"TBD", "TODO", "FIXME", "{{", "..."} {
		if strings.Contains(c, ph) {
			return "contains placeholder " + ph
		}
	}
	if len(c) < 20 {
		return "too short to mean anything"
	}
	if finds := spm.Scan(c, spm.Options{MinSeverity: spm.SevMajor}); len(finds) > 0 {
		return fmt.Sprintf("ambiguous at major severity (%q) — as empty as TBD", finds[0].Term)
	}
	return ""
}
