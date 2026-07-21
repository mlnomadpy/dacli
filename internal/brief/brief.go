// Package brief assembles the context document handed to a subagent.
//
// This is the product. Everything else in dacli exists so that this function
// has something to slice. Sections are emitted in fixed priority order and
// trimmed from the BOTTOM under a budget, so the highest-value content is
// never what gets cut; every omission is announced inline, because an agent
// can only ask for what it knows is missing.
package brief

import (
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/shortcut"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Options controls assembly.
type Options struct {
	Budget int // approximate token ceiling; 0 = unlimited
}

// Section is one emitted block. Order in the slice is priority order.
type Section struct {
	Title     string
	Content   string
	Droppable bool // the task itself is never droppable
}

// Brief is an assembled context document.
type Brief struct {
	TaskID   string
	Sections []Section
	Omitted  []string // announced omissions

	promptsDir string // workspace prompt-override dir, set by Assemble
}

// MillerCap bounds constraints and risks per brief. An agent handed 40
// constraints silently drops most of them, exactly as a human would; a brief
// is a working-memory budget, not an archive.
const MillerCap = 7

// EstimateTokens approximates token count. chars/4 is wrong per-model and
// every trim is announced anyway — the agent can see the estimate bit, which
// is most of the value a precise count would provide.
func EstimateTokens(s string) int { return len(s) / 4 }

// Assemble builds the brief for a task ref. Reads fold in pending events, so
// a sibling's finding is visible here the instant it is appended.
func Assemble(w *workspace.Workspace, ref string, opt Options) (*Brief, error) {
	t, err := store.FindTask(w, ref)
	if err != nil {
		return nil, err
	}
	p, err := store.LoadProject(w, t.Project)
	if err != nil {
		return nil, err
	}
	b := &Brief{TaskID: t.ID, promptsDir: w.PromptsDir()}

	// 1. The task itself — never trimmed. If it alone exceeds the budget,
	// assembly fails rather than truncating the one thing the agent needs.
	// The calibration line (PROPOSALS P2) rides here once history earns it:
	// n >= 10 completed estimate+actual pairs, never sooner — a multiplier
	// from three anecdotes is confidence theater.
	calib := ""
	if tp, ok := t.Estimate(); ok {
		if samples := store.CalibrationSamples(w); len(samples) >= 10 {
			ratios := make([]float64, len(samples))
			for i, s := range samples {
				ratios[i] = s.Ratio()
			}
			med := spm.Median(ratios)
			lo, hi := tp.ConfidenceRange(1)
			calib = fmt.Sprintf("calibrated: ~%.1f–%.1fh wall (×%.1f median, n=%d — time proxy, not tokens)",
				lo*med, hi*med, med, len(samples))
		}
	}
	b.add("Task: "+t.Title, taskSection(t, calib), false)

	// 2. Why — project goal chain, with the current lifecycle phase so the
	// agent knows what kind of work is appropriate NOW (don't implement in
	// discovery).
	var why strings.Builder
	fmt.Fprintf(&why, "Project **%s** — *%s*\n", p.Slug, p.Title)
	if phase, ok := p.Doc.Front.Get("phase"); ok && phase != "" {
		fmt.Fprintf(&why, "Phase: **%s**", phase)
		if allows := p.Doc.Front.GetList("phase_allows"); len(allows) > 0 {
			fmt.Fprintf(&why, " (work appropriate now: %s)", strings.Join(allows, ", "))
		}
		why.WriteString("\n")
	}
	if s, ok := p.Doc.Section("Goal"); ok && strings.TrimSpace(s.Content) != "" {
		why.WriteString("Goal: " + strings.TrimSpace(s.Content) + "\n")
	}
	if s, ok := p.Doc.Section("Success criteria"); ok && strings.TrimSpace(s.Content) != "" {
		why.WriteString("Success criteria:\n" + s.Content)
	}
	b.add("Why", why.String(), true)

	// 3. Scope boundary — cheap, and the only scope-creep intervention that
	// lands before the tokens are spent.
	if s, ok := p.Doc.Section("Out of scope"); ok && strings.TrimSpace(s.Content) != "" {
		b.add("Out of scope", s.Content, true)
	}

	// 3b. Codebase map — for adopted projects, the real structure of the
	// existing repo, so an agent onboards from context rather than a blank.
	if s, ok := p.Doc.Section("Codebase map"); ok && strings.TrimSpace(s.Content) != "" {
		b.add("Codebase map", s.Content, true)
	}

	// 4. Constraints — project constraints plus decision notes, capped.
	var cons strings.Builder
	if s, ok := p.Doc.Section("Constraints"); ok && strings.TrimSpace(s.Content) != "" {
		cons.WriteString(s.Content)
	}
	decisions, _ := store.ListNotes(w, p.Slug, model.NoteDecision)
	shown := 0
	for _, d := range decisions {
		if shown >= MillerCap {
			b.Omitted = append(b.Omitted, fmt.Sprintf("%d decisions beyond the working-memory cap", len(decisions)-shown))
			break
		}
		id, _ := d.Front.Get("id")
		fmt.Fprintf(&cons, "**[[%s]]**", id)
		if s, ok := d.Section("Chose"); ok {
			fmt.Fprintf(&cons, " — Chose: %s", strings.TrimSpace(s.Content))
		}
		if s, ok := d.Section("Rejected"); ok {
			fmt.Fprintf(&cons, " Rejected: %s.", strings.TrimSpace(s.Content))
		}
		if s, ok := d.Section("Because"); ok {
			fmt.Fprintf(&cons, " Because: %s", strings.TrimSpace(s.Content))
		}
		cons.WriteString("\n")
		shown++
	}
	if strings.TrimSpace(cons.String()) != "" {
		b.add("Constraints", cons.String(), true)
	}

	// 5. Risks — rank 1 and 2 only, WITH their indicators. A risk register
	// helps an agent only in this form: what is likely to go wrong, and what
	// the early warning looks like.
	risks, _ := store.ListRisks(w, p.Slug)
	var rk strings.Builder
	shownRisks := 0
	for _, r := range risks {
		if r.Rank() > 2 {
			continue // rank 3 is monitored, not briefed
		}
		if shownRisks >= MillerCap {
			b.Omitted = append(b.Omitted, "risks beyond the working-memory cap")
			break
		}
		fmt.Fprintf(&rk, "**%s** (rank %d)", r.Title, r.Rank())
		if len(r.Indicators) > 0 {
			fmt.Fprintf(&rk, " — watch for: %s", strings.Join(r.Indicators, "; "))
		}
		rk.WriteString("\n")
		shownRisks++
	}
	if rk.Len() > 0 {
		b.add("Risks", rk.String(), true)
	}

	// 6. Glossary — one definition per term for every agent in the tree.
	if g, err := mdstore.ReadFile(w.GlossaryPath(p.Slug)); err == nil {
		var body strings.Builder
		for _, s := range g.Sections {
			body.WriteString(s.Content)
		}
		if strings.TrimSpace(body.String()) != "" {
			b.add("Glossary", body.String(), true)
		}
	}

	// 7. Lessons — workspace-scoped notes from OTHER projects (PROPOSALS
	// P1): the compounding loop. Rendered quote-fenced like all third-party
	// content — lessons are data in briefs; only skills are instructions,
	// and the boundary between those is a security boundary (SKILLS.md § 6).
	lessons := store.WorkspaceLessons(w, p.Slug)
	if len(lessons) > 0 {
		var ls strings.Builder
		shown := 0
		for _, l := range lessons {
			if shown >= MillerCap {
				b.Omitted = append(b.Omitted, fmt.Sprintf("%d workspace lessons beyond the cap", len(lessons)-shown))
				break
			}
			writeQuoted(&ls, l.Actor+" · from "+l.Project, "", "[["+l.ID+"]] "+l.Title+" — "+l.Body)
			shown++
		}
		b.add("Lessons from other projects", ls.String(), true)
	}

	// 8. What siblings found — finding notes plus PENDING finding events, so
	// a report is visible tree-wide the instant it is written, no sync
	// needed. Third-party content is quote-fenced and attributed: data, not
	// instructions.
	var finds strings.Builder
	notes, _ := store.ListNotes(w, p.Slug, model.NoteFinding)
	for _, n := range notes {
		id, _ := n.Front.Get("id")
		by, _ := n.Front.Get("created_by")
		sev, _ := n.Front.Get("severity")
		// On disk the note's body lives inside the level-1 title section
		// (content extends to the next heading), so collect every section's
		// content — filtering by level here silently dropped finding bodies,
		// which the dogfood test caught on its first run.
		var body strings.Builder
		for _, s := range n.Sections {
			body.WriteString(s.Content)
		}
		writeQuoted(&finds, by, sev, "[["+id+"]] "+strings.TrimSpace(body.String()))
	}
	events, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventFinding}, Pending: true})
	for _, e := range events {
		if e.About != "" && e.About != t.ID && e.About != strings.TrimPrefix(t.ID, "t-") {
			continue
		}
		writeQuoted(&finds, e.Actor, "", e.Body)
	}
	if strings.TrimSpace(finds.String()) != "" {
		b.add("What siblings found", finds.String(), true)
	}

	// 9. Recent activity on this task.
	var act strings.Builder
	recent, _ := eventlog.List(w, eventlog.Query{About: t.ID, Limit: 5})
	for _, e := range recent {
		fmt.Fprintf(&act, "- %s %s by %s\n", e.ID[:10], e.Kind, e.Actor)
	}
	if act.Len() > 0 {
		b.add("Recent activity", act.String(), true)
	}

	// 10. Shortcuts — ranked by derived use count, truncated with the
	// omission announced. An unadvertised shortcut still runs; it just
	// stops taxing every brief.
	if scs, _ := store.LoadShortcuts(w); len(scs) > 0 {
		runs, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventRun}})
		counts := map[string]int{}
		for _, e := range runs {
			counts[e.About]++
		}
		for i := range scs {
			scs[i].Uses = counts[scs[i].Name]
		}
		if cat := shortcut.Catalog(scs, "", 8); strings.TrimSpace(cat) != "" {
			b.add("Shortcuts", cat, true)
		}
	}

	return b, b.trim(opt.Budget)
}

func (b *Brief) add(title, content string, droppable bool) {
	b.Sections = append(b.Sections, Section{Title: title, Content: content, Droppable: droppable})
}

// writeQuoted renders third-party content as an attributed blockquote — the
// cheap injection mitigation: it makes the provenance visible, not the
// attack impossible.
func writeQuoted(w *strings.Builder, by, severity, text string) {
	tag := by
	if severity != "" {
		tag += ", " + severity
	}
	fmt.Fprintf(w, "> **%s**:\n", tag)
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		fmt.Fprintf(w, "> %s\n", line)
	}
}

func taskSection(t *store.Task, calibLine string) string {
	var s strings.Builder
	meta := []string{}
	if p := t.Priority(); p != "" {
		meta = append(meta, "priority: "+p)
	}
	if tp, ok := t.Estimate(); ok {
		meta = append(meta, fmt.Sprintf("estimate: %g/%g/%g (Te %.1f)",
			tp.Optimistic, tp.Probable, tp.Pessimistic, tp.Expected()))
	}
	if o := t.Owner(); o != "" {
		meta = append(meta, "owner: "+o)
	}
	if len(meta) > 0 {
		s.WriteString(strings.Join(meta, " · ") + "\n")
	}
	if calibLine != "" {
		s.WriteString(calibLine + "\n")
	}
	if len(meta) > 0 || calibLine != "" {
		s.WriteString("\n")
	}
	for _, sec := range t.Doc.Sections {
		switch {
		case sec.Level == 1:
			// title already in the section header
		case strings.EqualFold(sec.Title, "Log"):
			// history, not context
		default:
			if strings.TrimSpace(sec.Content) == "" {
				continue
			}
			if sec.Title != "" {
				s.WriteString("**" + sec.Title + "**\n")
			}
			s.WriteString(sec.Content)
		}
	}
	return s.String()
}

// trim drops droppable sections from the bottom until the budget fits,
// announcing each drop. The task itself is never dropped: if it alone
// exceeds the budget, that is an error, not a truncation.
func (b *Brief) trim(budget int) error {
	if budget <= 0 {
		return nil
	}
	for EstimateTokens(b.render()) > budget {
		dropped := false
		for i := len(b.Sections) - 1; i >= 0; i-- {
			if b.Sections[i].Droppable {
				b.Omitted = append(b.Omitted, fmt.Sprintf("section %q (budget)", b.Sections[i].Title))
				b.Sections = append(b.Sections[:i], b.Sections[i+1:]...)
				dropped = true
				break
			}
		}
		if !dropped {
			return fmt.Errorf("task alone exceeds the %d-token budget; raise it — truncating the task would hand the agent half its instructions", budget)
		}
	}
	return nil
}

func (b *Brief) render() string {
	var s strings.Builder
	for _, sec := range b.Sections {
		s.WriteString("## " + sec.Title + "\n")
		s.WriteString(sec.Content)
		s.WriteString("\n")
	}
	return s.String()
}

// Render produces the final markdown document. The header prose is
// templated (prompts/tpl/brief_header.md, workspace-overridable) — the
// data-not-instructions line is a security posture and deserves review as a
// file, not as a string constant.
func (b *Brief) Render() string {
	var s strings.Builder
	header, err := prompts.Render(b.promptsDir, "brief_header", map[string]any{
		"TaskID": b.TaskID, "Est": EstimateTokens(b.render()),
	})
	if err != nil {
		// A broken header override degrades to the embedded default rather
		// than shipping a brief without the untrusted-content warning.
		header, _ = prompts.Render("", "brief_header", map[string]any{
			"TaskID": b.TaskID, "Est": EstimateTokens(b.render()),
		})
	}
	s.WriteString(strings.TrimRight(header, "\n") + "\n\n")
	s.WriteString(b.render())
	for _, o := range b.Omitted {
		fmt.Fprintf(&s, "<!-- dacli: omitted %s -->\n", o)
	}
	return s.String()
}
