package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

// project_sections: present-and-filled passes; missing or present-but-empty
// (or placeholder-bearing) fails and names which section and why.
func TestProjectSectionsPredicate(t *testing.T) {
	w, p := gateEnv(t)

	// "Goal" was created empty by CreateProject (gateEnv passes goal="").
	c := evaluate(w, p, Predicate{Kind: "project_sections", Arg: "Goal"})
	if c.OK {
		t.Fatal("an empty section must not satisfy the gate")
	}
	if !strings.Contains(c.Why, "Goal") || !strings.Contains(c.Why, "empty") {
		t.Errorf("Why must name the section and the reason; got %q", c.Why)
	}

	// A section that is present but too short to mean anything.
	p.Doc.SetSection("Goal", "short")
	if c := evaluate(w, p, Predicate{Kind: "project_sections", Arg: "Goal"}); c.OK {
		t.Error("a too-short section must not satisfy the gate")
	}

	// A section entirely absent from the document.
	if c := evaluate(w, p, Predicate{Kind: "project_sections", Arg: "Not A Real Section"}); c.OK {
		t.Error("a missing section must not satisfy the gate")
	} else if !strings.Contains(c.Why, "missing") {
		t.Errorf("Why must say the section is missing; got %q", c.Why)
	}

	// Filled: satisfies.
	p.Doc.SetSection("Goal", "Ship the export flag the whole team is blocked on this quarter.")
	if c := evaluate(w, p, Predicate{Kind: "project_sections", Arg: "Goal"}); !c.OK {
		t.Errorf("a genuinely filled section must satisfy the gate; Why=%q", c.Why)
	}

	// Multiple sections, one still unfilled.
	p.Doc.SetSection("Out of scope", "")
	c = evaluate(w, p, Predicate{Kind: "project_sections", Arg: "Goal | Out of scope"})
	if c.OK {
		t.Fatal("a mix with one unfilled section must not pass")
	}
	if !strings.Contains(c.Why, "Out of scope") {
		t.Errorf("Why must name the unfilled section; got %q", c.Why)
	}
	p.Doc.SetSection("Out of scope", "Publishing a tagged release remains out of scope.")
	if c := evaluate(w, p, Predicate{Kind: "project_sections", Arg: "Goal | Out of scope"}); !c.OK {
		t.Errorf("once every named section is filled, the gate must pass; Why=%q", c.Why)
	}
}

// glossary: min_terms N — below the floor fails, at/above it passes.
func TestGlossaryPredicate(t *testing.T) {
	w, p := gateEnv(t)

	c := evaluate(w, p, Predicate{Kind: "glossary", Arg: "min_terms 2"})
	if c.OK {
		t.Fatal("an empty glossary must not clear a nonzero floor")
	}
	if !strings.Contains(c.Why, "0 defined") {
		t.Errorf("Why must report the count; got %q", c.Why)
	}

	if err := store.GlossaryAdd(w, agentid.RootID, p.Slug, "gate", "a stage-transition check"); err != nil {
		t.Fatal(err)
	}
	if c := evaluate(w, p, Predicate{Kind: "glossary", Arg: "min_terms 2"}); c.OK {
		t.Fatal("one term must not clear a floor of two")
	}
	if err := store.GlossaryAdd(w, agentid.RootID, p.Slug, "cone", "the narrowing uncertainty band"); err != nil {
		t.Fatal(err)
	}
	c = evaluate(w, p, Predicate{Kind: "glossary", Arg: "min_terms 2"})
	if !c.OK {
		t.Errorf("two terms must clear a floor of two; Why=%q", c.Why)
	}
	if !strings.Contains(c.Why, "2 defined") {
		t.Errorf("Why must report the count; got %q", c.Why)
	}
}

// decisions: min N — every decision counted must carry a non-empty Rejected
// section; a decision note without one does not count toward the floor.
func TestDecisionsPredicate(t *testing.T) {
	w, p := gateEnv(t)

	c := evaluate(w, p, Predicate{Kind: "decisions", Arg: "min 1"})
	if c.OK {
		t.Fatal("zero decisions must not clear a floor of one")
	}

	// A decision note without a rejection cannot be created at all
	// (CreateNote refuses it) — confirm the refusal, then create one WITH a
	// rejection so the gate has something real to evaluate.
	if _, err := store.CreateNote(w, agentid.RootID, p.Slug, model.NoteDecision, "chose approach A", store.NoteOpts{}); err == nil {
		t.Fatal("a decision without --rejected must be refused at creation, not merely uncounted by the gate")
	}

	if _, err := store.CreateNote(w, agentid.RootID, p.Slug, model.NoteDecision, "chose approach A", store.NoteOpts{
		Rejected: "approach B", Because: "A is cheaper to reverse",
	}); err != nil {
		t.Fatal(err)
	}
	c = evaluate(w, p, Predicate{Kind: "decisions", Arg: "min 1"})
	if !c.OK {
		t.Errorf("a decision with a rejection must clear a floor of one; Why=%q", c.Why)
	}
	if !strings.Contains(c.Why, "1 of 1") {
		t.Errorf("Why must report the tally; got %q", c.Why)
	}
}

// tasks: all_have_acceptance — every task must carry acceptance criteria.
func TestTasksAllHaveAcceptancePredicate(t *testing.T) {
	w, p := gateEnv(t)

	if _, err := store.CreateTask(w, agentid.RootID, p.Slug, "no acceptance criteria", store.TaskOpts{}); err != nil {
		t.Fatal(err)
	}
	c := evaluate(w, p, Predicate{Kind: "tasks", Arg: "all_have_acceptance"})
	if c.OK {
		t.Fatal("a task with no acceptance criteria must not clear this gate")
	}
	if !strings.Contains(c.Why, "001") {
		t.Errorf("Why must name the offending task; got %q", c.Why)
	}

	if _, err := store.CreateTask(w, agentid.RootID, p.Slug, "has acceptance", store.TaskOpts{Accept: []string{"it works"}}); err != nil {
		t.Fatal(err)
	}
	c = evaluate(w, p, Predicate{Kind: "tasks", Arg: "all_have_acceptance"})
	if c.OK {
		t.Fatal("one bare task must still fail the gate even alongside a well-formed one")
	}
}

// tasks: all_have_estimate — every task must carry a three-point estimate.
func TestTasksAllHaveEstimatePredicate(t *testing.T) {
	w, p := gateEnv(t)

	if _, err := store.CreateTask(w, agentid.RootID, p.Slug, "unsized", store.TaskOpts{}); err != nil {
		t.Fatal(err)
	}
	if c := evaluate(w, p, Predicate{Kind: "tasks", Arg: "all_have_estimate"}); c.OK {
		t.Fatal("an unsized task must not clear this gate")
	}

	if _, err := store.CreateTask(w, agentid.RootID, p.Slug, "sized", store.TaskOpts{Estimate: "1,2,3"}); err != nil {
		t.Fatal(err)
	}
	if c := evaluate(w, p, Predicate{Kind: "tasks", Arg: "all_have_estimate"}); c.OK {
		t.Fatal("the unsized task from before must still shut the gate")
	}
}

// tasks: musts_done — every must-priority task must be status done.
func TestTasksMustsDonePredicate(t *testing.T) {
	w, p := gateEnv(t)

	must, err := store.CreateTask(w, agentid.RootID, p.Slug, "a must task", store.TaskOpts{Priority: "must"})
	if err != nil {
		t.Fatal(err)
	}
	if c := evaluate(w, p, Predicate{Kind: "tasks", Arg: "musts_done"}); c.OK {
		t.Fatal("an open must task must not clear this gate")
	}
	if err := store.CloseTask(w, must, agentid.RootID); err != nil {
		t.Fatal(err)
	}
	if c := evaluate(w, p, Predicate{Kind: "tasks", Arg: "musts_done"}); !c.OK {
		t.Errorf("once every must task is done the gate must pass; Why=%q", c.Why)
	}
}

// risks: rank1_have_action — every rank-1 (high impact, high likelihood) risk
// must carry a non-empty action plan.
func TestRisksRank1HaveActionPredicate(t *testing.T) {
	w, p := gateEnv(t)

	if _, err := store.CreateRisk(w, agentid.RootID, p.Slug, "vendor drift", model.LevelHigh, model.LevelHigh, nil, ""); err != nil {
		t.Fatal(err)
	}
	c := evaluate(w, p, Predicate{Kind: "risks", Arg: "rank1_have_action"})
	if c.OK {
		t.Fatal("a rank-1 risk with no action plan must not clear this gate")
	}
	if !strings.Contains(c.Why, "vendor-drift") {
		t.Errorf("Why must name the offending risk; got %q", c.Why)
	}

	if _, err := store.CreateRisk(w, agentid.RootID, p.Slug, "leak", model.LevelHigh, model.LevelHigh, nil, "watch for X"); err != nil {
		t.Fatal(err)
	}
	c = evaluate(w, p, Predicate{Kind: "risks", Arg: "rank1_have_action"})
	if c.OK {
		t.Fatal("the still-unactioned first risk must keep the gate shut")
	}

	// A rank-3 (low/low) risk needs no action plan at all — only rank-1 does.
	if _, err := store.CreateRisk(w, agentid.RootID, p.Slug, "monitor-only", model.LevelLow, model.LevelLow, nil, ""); err != nil {
		t.Fatal(err)
	}
	c = evaluate(w, p, Predicate{Kind: "risks", Arg: "rank1_have_action"})
	if c.OK {
		t.Fatal("still one unactioned rank-1 risk outstanding")
	}
}

// A read failure on the risks store must fail the gate closed, not read as
// "no risks" — the same vacuous-truth hazard the tasks gates already guard.
func TestRisksRank1PredicateFailsClosedOnUnreadableSet(t *testing.T) {
	w, p := gateEnv(t)
	risksDir := w.RisksDir(p.Slug)
	if err := os.MkdirAll(filepath.Dir(risksDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(risksDir, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := evaluate(w, p, Predicate{Kind: "risks", Arg: "rank1_have_action"})
	if c.OK {
		t.Fatal("risks gate passed on an unreadable risk set — a gate must never certify what it could not read")
	}
	if !strings.Contains(strings.ToLower(c.Why), "could not read") {
		t.Errorf("Why must name the read failure; got %q", c.Why)
	}
}

// retro: a "Retro:" heading inside a ref note satisfies; anything else does not.
func TestRetroPredicate(t *testing.T) {
	w, p := gateEnv(t)

	if c := evaluate(w, p, Predicate{Kind: "retro", Arg: "required"}); c.OK {
		t.Fatal("no retro recorded yet must not clear this gate")
	}

	// An unrelated ref note must not satisfy it — the heading must be a retro.
	if _, err := store.CreateNote(w, agentid.RootID, p.Slug, model.NoteRef, "not a retro", store.NoteOpts{Body: "irrelevant"}); err != nil {
		t.Fatal(err)
	}
	if c := evaluate(w, p, Predicate{Kind: "retro", Arg: "required"}); c.OK {
		t.Fatal("a ref note with no Retro: heading must not satisfy the gate")
	}

	if _, err := store.CreateNote(w, agentid.RootID, p.Slug, model.NoteRef, "Retro: sprint N", store.NoteOpts{Body: "what we'd do differently"}); err != nil {
		t.Fatal(err)
	}
	if c := evaluate(w, p, Predicate{Kind: "retro", Arg: "required"}); !c.OK {
		t.Errorf("a recorded retro must satisfy the gate; Why=%q", c.Why)
	}
}

// A predicate kind the running build does not recognize must refuse, not
// vacuously pass — the manifest and the binary can disagree after either one
// is edited alone.
func TestUnknownPredicateKindRefuses(t *testing.T) {
	w, p := gateEnv(t)
	c := evaluate(w, p, Predicate{Kind: "wat", Arg: "??"})
	if c.OK {
		t.Fatal("an unrecognized predicate kind must never pass")
	}
	if !strings.Contains(c.Why, "unknown predicate") {
		t.Errorf("Why must say the predicate is unknown; got %q", c.Why)
	}

	// An unrecognized sub-argument on a known kind falls through the same way.
	if c := evaluate(w, p, Predicate{Kind: "tasks", Arg: "not_a_real_check"}); c.OK {
		t.Fatal("an unrecognized tasks sub-check must never pass")
	}
	if c := evaluate(w, p, Predicate{Kind: "risks", Arg: "not_a_real_check"}); c.OK {
		t.Fatal("an unrecognized risks sub-check must never pass")
	}
}
