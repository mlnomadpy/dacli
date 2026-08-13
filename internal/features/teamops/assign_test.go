package teamops

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func assignEnv(t *testing.T) (*clikit.Ctx, *workspace.Workspace, *bytes.Buffer) {
	t.Helper()
	unsetAgentEnv(t)
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "P", "p", "", ""); err != nil {
		t.Fatal(err)
	}
	for _, r := range []team.Role{
		{Name: "junior", Kind: "implementer", Runtime: "generic", Profile: team.ModelProfile{ID: "small", CostTier: 1, MaxTaskPoints: 3}, Summary: "small work"},
		{Name: "fixer", Kind: "implementer", Runtime: "generic", Profile: team.ModelProfile{ID: "medium", CostTier: 2, MaxTaskPoints: 8}, Summary: "normal work"},
		{Name: "maintainer", Kind: "implementer", Runtime: "generic", Profile: team.ModelProfile{ID: "large", CostTier: 3}, Summary: "heavy work"},
		{Name: "reviewer", Kind: "reviewer", Runtime: "generic", Profile: team.ModelProfile{ID: "large", CostTier: 3}, Summary: "reviews"},
	} {
		if err := store.CreateRole(w, agentid.RootID, r); err != nil {
			t.Fatal(err)
		}
	}
	out := &bytes.Buffer{}
	return &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root}, w, out
}

func sized(t *testing.T, w *workspace.Workspace, title, est string) *store.Task {
	t.Helper()
	tk, err := store.CreateTask(w, agentid.RootID, "p", title, store.TaskOpts{Accept: []string{"ok"}, Estimate: est})
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

// Difficulty should pick the model. dacli already carried a per-role model tier
// and a capacity cap, and the seniority gate already refused a role that was too
// small — but nothing ever CHOSE one, so cheap models were only used when an
// operator remembered to ask for them (dacli 231).
func TestTeamAssignRoutesBySize(t *testing.T) {
	ctx, w, out := assignEnv(t)

	for _, tc := range []struct{ est, wantRole, wantModel string }{
		{"1,2,3", "junior", "small"},       // Te 2 — cheap model holds it
		{"3,5,7", "fixer", "medium"},       // Te 5 — past junior's cap
		{"8,12,20", "maintainer", "large"}, // Te ~12.7 — only the uncapped role fits
	} {
		out.Reset()
		tk := sized(t, w, "work "+tc.est, tc.est)
		if err := cmdTeamAssign(ctx, []string{tk.Slug}); err != nil {
			t.Fatalf("estimate %s: %v", tc.est, err)
		}
		got := out.String()
		if !strings.Contains(got, tc.wantRole) {
			t.Errorf("estimate %s routed to the wrong role; want %s, got:\n%s", tc.est, tc.wantRole, got)
		}
		if !strings.Contains(got, tc.wantModel) {
			t.Errorf("estimate %s should name the model %s, got:\n%s", tc.est, tc.wantModel, got)
		}
	}
}

func TestTeamAssignPrintsRuntimeModelAndDecisionFactors(t *testing.T) {
	ctx, w, out := assignEnv(t)
	tk := sized(t, w, "work with a declared profile", "3,5,7")
	if err := cmdTeamAssign(ctx, []string{tk.Slug}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"runtime", "model", "cost tier", "task capacity"} {
		if !strings.Contains(got, want) {
			t.Errorf("assignment must print decision factor %q, got:\n%s", want, got)
		}
	}
}

// An unsized task cannot be routed by capacity. Saying so beats silently
// defaulting to the expensive role, which is how cost leaks.
func TestTeamAssignRefusesAnUnsizedTask(t *testing.T) {
	ctx, w, _ := assignEnv(t)
	tk, err := store.CreateTask(w, agentid.RootID, "p", "unsized", store.TaskOpts{Accept: []string{"ok"}})
	if err != nil {
		t.Fatal(err)
	}
	err = cmdTeamAssign(ctx, []string{tk.Slug})
	if err == nil {
		t.Fatal("an unsized task must not be routed")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit %d, want 3 (refused)", clikit.ExitCode(err))
	}
	if !strings.Contains(err.Error(), "task estimate") {
		t.Errorf("the refusal should name the fix; got: %v", err)
	}
}

// --kind asks a different question ("who would review this") without changing
// the task or the phase.
func TestTeamAssignHonorsExplicitKind(t *testing.T) {
	ctx, w, out := assignEnv(t)
	tk := sized(t, w, "reviewable", "1,2,3")
	if err := cmdTeamAssign(ctx, []string{tk.Slug, "--kind", "reviewer"}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "reviewer") {
		t.Errorf("--kind reviewer must select a reviewer role, got:\n%s", got)
	}
}

// No role of that kind can hold the work: refuse and say what to do, rather
// than picking something that the seniority gate would then reject at spawn.
func TestTeamAssignRefusesWhenNothingFits(t *testing.T) {
	ctx, w, _ := assignEnv(t)
	tk := sized(t, w, "enormous", "40,60,90")
	err := cmdTeamAssign(ctx, []string{tk.Slug, "--kind", "designer"})
	if err == nil {
		t.Fatal("no designer role exists; assign must refuse")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit %d, want 3 (refused)", clikit.ExitCode(err))
	}
}

// With no --kind, a verb in the title names the work: "review …" routes to a
// reviewer even while the project is still implementing, and the inference is
// printed so a wrong guess is visible. Before dacli 265 this silently defaulted
// to implementer.
func TestTeamAssignInfersKindFromTitleVerb(t *testing.T) {
	ctx, w, out := assignEnv(t)
	tk := sized(t, w, "review the burn alert threshold", "1,2,3")
	if err := cmdTeamAssign(ctx, []string{tk.Slug}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "reviewer") {
		t.Errorf("a 'review …' title should route to a reviewer, got:\n%s", got)
	}
	if !strings.Contains(got, "title verb") {
		t.Errorf("the inferred kind and its source must be printed, got:\n%s", got)
	}
}

// A task that declares its own role_kind beats any title guess.
func TestTeamAssignInfersKindFromTaskField(t *testing.T) {
	ctx, w, out := assignEnv(t)
	tk := sized(t, w, "make it faster", "1,2,3") // title has no verb hint
	tk.Doc.Front.Set("role_kind", "reviewer")
	if err := store.SaveTask(tk); err != nil {
		t.Fatal(err)
	}
	if err := cmdTeamAssign(ctx, []string{tk.Slug}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "reviewer") {
		t.Errorf("a declared role_kind must route by that kind, got:\n%s", got)
	}
	if !strings.Contains(got, "role_kind") {
		t.Errorf("the source of the inference must be printed, got:\n%s", got)
	}
}

// Nothing in the task or phase speaks up: implement it, and say so.
func TestTeamAssignDefaultsToImplementerAndPrintsIt(t *testing.T) {
	ctx, w, out := assignEnv(t)
	tk := sized(t, w, "make it faster", "1,2,3")
	if err := cmdTeamAssign(ctx, []string{tk.Slug}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "kind implementer (default)") {
		t.Errorf("an unhinted task should default to implementer and print the source, got:\n%s", got)
	}
}

// An explicit --kind is labelled as such, not as an inference.
func TestTeamAssignLabelsExplicitKind(t *testing.T) {
	ctx, w, out := assignEnv(t)
	tk := sized(t, w, "reviewable", "1,2,3")
	if err := cmdTeamAssign(ctx, []string{tk.Slug, "--kind", "reviewer"}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "explicit --kind") {
		t.Errorf("an explicit --kind must not read as an inference, got:\n%s", got)
	}
}

func TestTeamAssignUsageAndNotFound(t *testing.T) {
	ctx, w, _ := assignEnv(t)
	sized(t, w, "some work", "1,2,3")

	if err := cmdTeamAssign(ctx, nil); clikit.ExitCode(err) != 2 {
		t.Errorf("no ref: exit %d, want 2", clikit.ExitCode(err))
	}
	if err := cmdTeamAssign(ctx, []string{"999"}); clikit.ExitCode(err) != 4 {
		t.Errorf("unknown ref: exit %d, want 4", clikit.ExitCode(err))
	}
	if err := cmdTeamAssign(ctx, []string{"001", "--kindd", "implementer"}); clikit.ExitCode(err) != 2 {
		t.Errorf("typo'd flag: exit %d, want 2", clikit.ExitCode(err))
	}
}
