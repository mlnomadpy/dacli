package gates

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Attach binds a template, sets the project to its first stage and cone, and
// writes the phase so brief/spawn can read it cheaply.
func TestAttachSetsFirstStageConeAndPhase(t *testing.T) {
	w, p := gateEnv(t)

	first, err := Attach(w, p.Slug, "product")
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "discovery" {
		t.Errorf("Attach returned stage %q, want discovery", first.Name)
	}

	reloaded, err := store.LoadProject(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := reloaded.Doc.Front.Get("template"); v != "product" {
		t.Errorf("template front-matter = %q, want product", v)
	}
	if v, _ := reloaded.Doc.Front.Get("template_stage"); v != "discovery" {
		t.Errorf("template_stage front-matter = %q, want discovery", v)
	}
	if v, _ := reloaded.Doc.Front.Get("stage"); v != "definition" {
		t.Errorf("cone (stage) front-matter = %q, want definition", v)
	}
	if v, _ := reloaded.Doc.Front.Get("phase"); v != "discovery" {
		t.Errorf("phase front-matter = %q, want discovery", v)
	}
	if allows := reloaded.Doc.Front.GetList("phase_allows"); len(allows) != 2 || allows[0] != "researcher" {
		t.Errorf("phase_allows = %v, want [researcher reviewer]", allows)
	}
}

// A zero-stage template (solo) attaches as already-complete and writes no
// phase gate — most work should not pay for process.
func TestAttachSoloTemplateIsImmediatelyComplete(t *testing.T) {
	w, p := gateEnv(t)

	stage, err := Attach(w, p.Slug, "solo")
	if err != nil {
		t.Fatal(err)
	}
	if stage.Name != "complete" {
		t.Errorf("solo attach returned stage %q, want complete", stage.Name)
	}
	st, err := Status(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Complete {
		t.Error("a project attached to solo must read as Complete")
	}

	ph, err := PhaseFor(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if ph.Gated {
		t.Error("a solo-attached project must not be phase-gated")
	}
	if !ph.AllowsKind("implementer") {
		t.Error("an ungated phase must allow every role kind")
	}
}

// Attach on an unknown template name must error, not attach garbage.
func TestAttachUnknownTemplateErrors(t *testing.T) {
	w, p := gateEnv(t)
	if _, err := Attach(w, p.Slug, "does-not-exist"); err == nil {
		t.Fatal("attaching an unknown template must error")
	}
}

// Status on an untemplated (fresh) project reads as Complete/solo — no
// gate blocks work that never opted into one.
func TestStatusOnUntemplatedProjectIsComplete(t *testing.T) {
	w, p := gateEnv(t)
	st, err := Status(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Complete {
		t.Error("an untemplated project must read as Complete")
	}
	if len(st.Checks) != 0 {
		t.Errorf("an untemplated project must carry no checks; got %v", st.Checks)
	}
}

// Status names the project's evaluated checks for its current stage.
func TestStatusEvaluatesCurrentStageChecks(t *testing.T) {
	w, p := gateEnv(t)
	if _, err := Attach(w, p.Slug, "product"); err != nil {
		t.Fatal(err)
	}
	st, err := Status(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if st.Complete {
		t.Fatal("a freshly attached multi-stage template must not read as complete")
	}
	if len(st.Checks) == 0 {
		t.Fatal("Status must evaluate the current stage's predicates")
	}
	if st.Next == nil || st.Next.Name != "research" {
		t.Errorf("Next = %v, want the research stage", st.Next)
	}
}

// If the manifest changes under a project (its recorded stage name is no
// longer defined), Status must error rather than silently pick a stage.
func TestStatusErrorsWhenManifestNoLongerDefinesTheRecordedStage(t *testing.T) {
	w, p := gateEnv(t)
	if _, err := Attach(w, p.Slug, "product"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.LoadProject(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	reloaded.Doc.Front.Set("template_stage", "a-stage-the-manifest-has-never-heard-of")
	if err := store.SaveProject(reloaded); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(w, p.Slug); err == nil {
		t.Fatal("Status must error when the project's stage is not in the template")
	}
}

// Advance refuses (returns unmet, no error, no mutation) while a check is
// unmet, and moves the project forward once every check passes.
func TestAdvanceRefusesUntilChecksPassThenMoves(t *testing.T) {
	w, p := gateEnv(t)
	if _, err := Attach(w, p.Slug, "product"); err != nil {
		t.Fatal(err)
	}

	newStage, unmet, err := Advance(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if newStage != "" || len(unmet) == 0 {
		t.Fatalf("Advance must refuse with unmet checks on an unfilled discovery stage; newStage=%q unmet=%v", newStage, unmet)
	}

	// Satisfy discovery's gate: project_sections Goal|Out of scope, glossary min_terms 3.
	reloaded, err := store.LoadProject(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	reloaded.Doc.SetSection("Goal", "Ship the export flag the whole team is blocked on this quarter.")
	reloaded.Doc.SetSection("Out of scope", "Publishing a tagged release remains out of scope for now.")
	if err := store.SaveProject(reloaded); err != nil {
		t.Fatal(err)
	}
	for _, term := range []string{"gate", "cone", "phase"} {
		if err := store.GlossaryAdd(w, agentid.RootID, p.Slug, term, "defined for the test"); err != nil {
			t.Fatal(err)
		}
	}

	newStage, unmet, err = Advance(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if len(unmet) != 0 {
		t.Fatalf("Advance still refused once every check passed: %v", unmet)
	}
	if newStage != "research" {
		t.Errorf("Advance moved to %q, want research", newStage)
	}

	reloaded, err = store.LoadProject(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := reloaded.Doc.Front.Get("template_stage"); v != "research" {
		t.Errorf("template_stage on disk = %q, want research", v)
	}
	if v, _ := reloaded.Doc.Front.Get("phase"); v != "research" {
		t.Errorf("phase on disk = %q, want research", v)
	}
}

// Advance on an already-complete project is a no-op success, not an error.
func TestAdvanceOnCompleteProjectIsNoop(t *testing.T) {
	w, p := gateEnv(t)
	newStage, unmet, err := Advance(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if newStage != "complete" || len(unmet) != 0 {
		t.Errorf("Advance on an untemplated project = (%q, %v), want (complete, nil)", newStage, unmet)
	}
}

// Advancing past the LAST stage marks the project complete and clears the
// phase gate (writePhase's empty-phase branch).
func TestAdvancePastLastStageCompletesAndClearsPhase(t *testing.T) {
	w, p := gateEnv(t)
	if _, err := Attach(w, p.Slug, "research-paper"); err != nil {
		t.Fatal(err)
	}
	// research-paper's stages carry no phase/allow lines, so attaching must
	// not have set one — confirms the empty branch of writePhase at Attach too.
	ph, err := PhaseFor(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if ph.Gated {
		t.Fatal("research-paper defines no phase lines; the project must not be phase-gated")
	}

	tmpl, err := Get(w, "research-paper")
	if err != nil {
		t.Fatal(err)
	}
	for range tmpl.Stages {
		if err := satisfyCurrentStage(t, w, p.Slug); err != nil {
			t.Fatal(err)
		}
		newStage, unmet, err := Advance(w, p.Slug)
		if err != nil {
			t.Fatal(err)
		}
		if len(unmet) != 0 {
			t.Fatalf("Advance refused after satisfying every predicate: %v", unmet)
		}
		_ = newStage
	}

	st, err := Status(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Complete {
		t.Error("advancing past the final stage must mark the project complete")
	}
}

// satisfyCurrentStage makes every predicate of the project's current stage
// pass, regardless of which template it is on — used to drive Advance
// through a template's full stage sequence without hand-coding each gate.
func satisfyCurrentStage(t *testing.T, w *workspace.Workspace, slug string) error {
	t.Helper()
	p, err := store.LoadProject(w, slug)
	if err != nil {
		return err
	}
	tmplName, _ := p.Doc.Front.Get("template")
	stageName, _ := p.Doc.Front.Get("template_stage")
	tmpl, err := Get(w, tmplName)
	if err != nil {
		return err
	}
	for _, s := range tmpl.Stages {
		if s.Name != stageName {
			continue
		}
		for _, pred := range s.Predicates {
			switch pred.Kind {
			case "project_sections":
				for _, name := range strings.Split(pred.Arg, "|") {
					p.Doc.SetSection(strings.TrimSpace(name), "Real content describing this section, well past the minimum length floor.")
				}
			case "glossary":
				if err := store.GlossaryAdd(w, agentid.RootID, slug, "term-for-stage", "a definition long enough to count"); err != nil {
					return err
				}
				if err := store.GlossaryAdd(w, agentid.RootID, slug, "second-term", "another definition"); err != nil {
					return err
				}
				if err := store.GlossaryAdd(w, agentid.RootID, slug, "third-term", "a third definition"); err != nil {
					return err
				}
			case "decisions":
				if _, err := store.CreateNote(w, agentid.RootID, slug, model.NoteDecision,
					"stage decision", store.NoteOpts{Rejected: "the alternative", Because: "it was worse"}); err != nil {
					return err
				}
			case "tasks":
				switch pred.Arg {
				case "all_have_acceptance", "all_have_estimate":
					if _, err := store.CreateTask(w, agentid.RootID, slug, "well-formed task", store.TaskOpts{
						Accept: []string{"it works"}, Estimate: "1,2,3",
					}); err != nil {
						return err
					}
				case "musts_done":
					// no open must task exists yet in this project — vacuously satisfied
				}
			case "risks":
				if _, err := store.CreateRisk(w, agentid.RootID, slug, "a stage risk", model.LevelHigh, model.LevelHigh, nil, "watch and mitigate"); err != nil {
					return err
				}
			case "retro":
				if _, err := store.CreateNote(w, agentid.RootID, slug, model.NoteRef,
					"Retro: stage retro", store.NoteOpts{Body: "what we learned"}); err != nil {
					return err
				}
			}
		}
		return store.SaveProject(p)
	}
	return nil
}

// Stage.AllowsKind and Phase.AllowsKind: an empty Allow list is permissive; a
// non-empty one gates strictly, and an actor with no kind always passes
// (phase gating is opt-in per role).
func TestAllowsKindGating(t *testing.T) {
	open := Stage{}
	if !open.AllowsKind("anything") {
		t.Error("an empty Allow list must permit every kind")
	}
	gated := Stage{Allow: []string{"researcher", "reviewer"}}
	if !gated.AllowsKind("researcher") {
		t.Error("a listed kind must be allowed")
	}
	if gated.AllowsKind("implementer") {
		t.Error("an unlisted kind must not be allowed")
	}
	if !gated.AllowsKind("") {
		t.Error("a role with no kind must always pass (phase gating is opt-in per role)")
	}

	ungatedPhase := Phase{Gated: false, Allows: []string{"researcher"}}
	if !ungatedPhase.AllowsKind("implementer") {
		t.Error("an ungated Phase must permit every kind regardless of Allows")
	}
	gatedPhase := Phase{Gated: true, Allows: []string{"researcher"}}
	if !gatedPhase.AllowsKind("researcher") {
		t.Error("a gated Phase must allow a listed kind")
	}
	if gatedPhase.AllowsKind("implementer") {
		t.Error("a gated Phase must refuse an unlisted kind")
	}
	permissiveGatedPhase := Phase{Gated: true}
	if !permissiveGatedPhase.AllowsKind("implementer") {
		t.Error("a gated Phase with an empty Allows list must be permissive")
	}
}

// PhaseFor on a project that was never attached (or has no phase line, e.g.
// a template stage with no phase:) must report Gated=false, not error.
func TestPhaseForUngatedProject(t *testing.T) {
	w, p := gateEnv(t)
	ph, err := PhaseFor(w, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if ph.Gated {
		t.Error("a fresh project with no phase recorded must not be Gated")
	}
}

// unfilled: the present-vs-filled rule directly, across every reason it can
// refuse — empty, placeholder, too short, ambiguous — and the genuinely
// filled case that returns "".
func TestUnfilled(t *testing.T) {
	cases := []struct {
		name      string
		content   string
		wantEmpty bool
	}{
		{"empty", "", false},
		{"whitespace-only", "   \n\t  ", false},
		{"TBD placeholder", "TBD", false},
		{"TODO placeholder", "some notes, TODO: fill in later, more notes here", false},
		{"too short", "short", false},
		{"filled", "The export flag ships once every task under it clears its acceptance criteria.", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := unfilled(c.content)
			if (got == "") != c.wantEmpty {
				t.Errorf("unfilled(%q) = %q, wantEmpty=%v", c.content, got, c.wantEmpty)
			}
		})
	}
}
