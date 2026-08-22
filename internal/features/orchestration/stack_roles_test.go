package orchestration

// Stack-aware loop role defaults (dacli 192). The loop used to hand every
// project the constants "fixer" and "go-auditor" — a Python app was reviewed by
// a role named for a language it does not contain. These tests pin both halves:
// a recorded stack moves the default onto a roster role that exists, and a
// project with nothing recorded stays on the pre-192 constants exactly.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestVueSupabaseLoopDryRunRoutesImplementationAndReviewSeats(t *testing.T) {
	w := loopEnv(t)
	for _, path := range []string{"web/src/App.vue", "supabase/migrations/001.sql"} {
		full := filepath.Join(w.Root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	recordStack(t, w, "p", "Stack: Vue/Supabase. Build with `npm run build`; test with `npm test`.\n")
	roles := []team.Role{
		{Name: "fixer", Kind: "implementer", Grant: "rw", Runtime: "codex", Summary: "General implementation work", Profile: team.ModelProfile{ID: "mini", CostTier: 1, MaxTaskPoints: 3}},
		{Name: "frontend-fixer", Kind: "implementer", Grant: "rw", Runtime: "codex", Summary: "Vue UI frontend work", Scope: []string{"web/**"}, Profile: team.ModelProfile{ID: "mini", CostTier: 1, MaxTaskPoints: 3}},
		{Name: "supabase-junior", Kind: "implementer", Grant: "rw", Runtime: "codex", Summary: "Supabase SQL work", Scope: []string{"supabase/**"}, Profile: team.ModelProfile{ID: "mini", CostTier: 1, MaxTaskPoints: 3}},
		{Name: "supabase-implementer", Kind: "implementer", Grant: "rw", Runtime: "claude", Summary: "Supabase transactional SQL and database migrations", Scope: []string{"supabase/**"}, Profile: team.ModelProfile{ID: "sonnet", CostTier: 2, MaxTaskPoints: 8}},
		{Name: "frontend-reviewer", Kind: "reviewer", Grant: "ro", Runtime: "claude", Summary: "Reviews Vue frontend and Supabase integration", Profile: team.ModelProfile{ID: "sonnet", CostTier: 2}},
		{Name: "go-auditor", Kind: "reviewer", Grant: "ro", Runtime: "codex", Summary: "Reviews Go code", Scope: []string{"**/*.go"}, Profile: team.ModelProfile{ID: "mini", CostTier: 1}},
	}
	for _, role := range roles {
		if err := store.CreateRole(w, "a-root", role); err != nil {
			t.Fatal(err)
		}
	}
	ui, err := store.CreateTask(w, "a-root", "p", "Fix bounded Vue UI in web/src/App.vue", store.TaskOpts{Accept: []string{"UI works"}, Estimate: "1,1,1"})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := store.CreateTask(w, "a-root", "p", "Implement transactional SQL in supabase/migrations/001.sql", store.TaskOpts{Accept: []string{"transaction is atomic"}, Estimate: "2,2,2"})
	if err != nil {
		t.Fatal(err)
	}

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3})
	d.cfg.dryRun = true
	d.cfg.width = 2
	d.cfg.reviewRole = stackReviewRole(w, loopStack(w, "p"), "go-auditor")
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}
	if got := spawnRoleForTask(buildSpawnCalls(fr), ui.ID); got != "frontend-fixer" {
		t.Fatalf("Vue UI role = %q, want frontend-fixer", got)
	}
	if got := spawnRoleForTask(buildSpawnCalls(fr), sql.ID); got != "supabase-implementer" {
		t.Fatalf("transactional SQL role = %q, want consequence-uplifted supabase-implementer", got)
	}
	for _, call := range fr.calls {
		if len(call) > 0 && call[0] == "spawn" && contains(call, "go-auditor") {
			t.Fatalf("Vue/Supabase review selected go-auditor: %v", call)
		}
	}
	out := d.ctx.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{"frontend-fixer", "runtime codex", "model mini", "supabase-implementer", "consequence uplift", "frontend-reviewer", "source project stack"} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run output missing %q:\n%s", want, out)
		}
	}
}

func TestLoopDoesNotBypassAutomaticRoutingRefusal(t *testing.T) {
	w := loopEnv(t)
	if err := store.CreateRole(w, "a-root", team.Role{
		Name: "fixer", Kind: "implementer", Grant: "ro", Runtime: "codex",
		Profile: team.ModelProfile{ID: "mini", CostTier: 1, MaxTaskPoints: 8},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTask(w, "a-root", "p", "Fix implementation", store.TaskOpts{Accept: []string{"done"}, Estimate: "1,1,1"}); err != nil {
		t.Fatal(err)
	}

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3})
	d.cfg.dryRun = true
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}
	if calls := buildSpawnCalls(fr); len(calls) != 0 {
		t.Fatalf("routing refusal was bypassed by fallback spawn: %v", calls)
	}
	if out := d.ctx.Stdout.(*bytes.Buffer).String(); !strings.Contains(out, "found no eligible implementer role") {
		t.Fatalf("routing refusal was not explained:\n%s", out)
	}
}

// recordStack writes the stack the way `dacli new` writes it, so this test
// exercises the same prose the real command produces rather than a shape
// invented for the test.
func recordStack(t *testing.T, w *workspace.Workspace, slug, body string) {
	t.Helper()
	p, err := store.LoadProject(w, slug)
	if err != nil {
		t.Fatal(err)
	}
	p.Doc.SetSection("Constraints", body)
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
}

func rosterLookup(w *workspace.Workspace) func(string) bool {
	return func(name string) bool { _, ok := store.LoadRole(w, name); return ok }
}

func TestLoopRolesFollowRecordedStack(t *testing.T) {
	w := loopEnv(t)
	recordStack(t, w, "p", "Stack: Python. Build with `python -m build`; test with `pytest`. A task in this project is done only when both exit 0.\n")
	if err := store.CreateRole(w, "a-root", team.Role{
		Name: "python-auditor", Summary: "Audits Python for correctness and style.", Kind: "reviewer",
	}); err != nil {
		t.Fatal(err)
	}

	s := loopStack(w, "p")
	if s.Key != "python" {
		t.Fatalf("stack not read back from the project doc: %+v", s)
	}
	exists := rosterLookup(w)
	if got := prompts.RoleFor(s, "auditor", "go-auditor", exists); got != "python-auditor" {
		t.Errorf("review role = %q, want python-auditor", got)
	}
	// No python-fixer in the roster, so the impl default holds — the loop must
	// never spawn a role name that does not exist.
	if got := prompts.RoleFor(s, "fixer", "fixer", exists); got != "fixer" {
		t.Errorf("impl role = %q, want fixer", got)
	}
}

func TestLoopRolesUnchangedWithoutRecordedStack(t *testing.T) {
	w := loopEnv(t) // loopEnv's project records no stack, like every pre-192 one
	if err := store.CreateRole(w, "a-root", team.Role{
		Name: "python-auditor", Summary: "Audits Python for correctness and style.", Kind: "reviewer",
	}); err != nil {
		t.Fatal(err)
	}

	s := loopStack(w, "p")
	if s.Recorded() {
		t.Fatalf("a stackless project produced a stack: %+v", s)
	}
	exists := rosterLookup(w)
	if got := prompts.RoleFor(s, "auditor", "go-auditor", exists); got != "go-auditor" {
		t.Errorf("review role = %q, want go-auditor (pre-192 default)", got)
	}
	if got := prompts.RoleFor(s, "fixer", "fixer", exists); got != "fixer" {
		t.Errorf("impl role = %q, want fixer (pre-192 default)", got)
	}
	// A project that does not exist at all is the same answer, not a panic.
	if missing := loopStack(w, "no-such-project"); missing.Recorded() {
		t.Errorf("missing project produced a stack: %+v", missing)
	}
}
