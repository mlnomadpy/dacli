package orchestration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
)

// TestWaveSelectionBackfillsPastClaimCollision reproduces issue #838 through
// both public planners. Three equally critical ready tasks are ranked in file
// order: the first two claim the same feature slice and the third is
// independent. Width two must mean two compatible workers, not "inspect two
// candidates and spend one slot on a collision". The profile preview and loop
// execution preview must therefore choose the same first and third tasks. A
// skipped collision must never reach spawn, which is the boundary that creates
// an agent identity and worktree.
func TestWaveSelectionBackfillsPastClaimCollision(t *testing.T) {
	w := loopEnv(t)
	setProjectCodebaseMap(t, w, "Go")
	if err := store.CreateRuntime(w, "a-root", store.Runtime{Name: "wave-codex", Harness: "codex", Binary: "agent"}, "issue 838 single-harness fixture"); err != nil {
		t.Fatal(err)
	}
	for _, role := range []team.Role{
		{Name: "wave-fixer", Kind: "implementer", Grant: "rw", Runtime: "wave-codex", Profile: team.ModelProfile{ID: "wave-model", CostTier: 1, MaxTaskPoints: 8}},
		{Name: "wave-reviewer", Kind: "reviewer", Grant: "ro", Runtime: "wave-codex", Profile: team.ModelProfile{ID: "wave-review", CostTier: 1}},
	} {
		if err := store.CreateRole(w, "a-root", role); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{
		"internal/features/execution/fixture.go",
		"docs/independent.md",
	} {
		full := filepath.Join(w.Root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	newTask := func(title string) *store.Task {
		t.Helper()
		task, err := store.CreateTask(w, "a-root", "p", title, store.TaskOpts{
			Priority: "must", Estimate: "3,3,3", Accept: []string{"observable result"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return task
	}
	first := newTask("Fix first defect in internal/features/execution")
	rejected := newTask("Fix second defect in internal/features/execution")
	independent := newTask("Document independent behavior in docs/independent.md")
	if _, _, overlap := procmon.PathsOverlap(store.ClaimHints(w.Root, first), store.ClaimHints(w.Root, rejected)); !overlap {
		t.Fatalf("test setup: first claims %v do not overlap rejected claims %v", store.ClaimHints(w.Root, first), store.ClaimHints(w.Root, rejected))
	}
	if _, _, overlap := procmon.PathsOverlap(store.ClaimHints(w.Root, first), store.ClaimHints(w.Root, independent)); overlap {
		t.Fatalf("test setup: independent claims %v overlap first claims %v", store.ClaimHints(w.Root, independent), store.ClaimHints(w.Root, first))
	}

	profileOut := &bytes.Buffer{}
	profileCtx := &clikit.Ctx{Stdout: profileOut, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdStart(profileCtx, []string{"--project", "p", "--profile", "wave", "--width", "2", "--harness", "codex", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	var plan ProfilePlan
	if err := json.Unmarshal(profileOut.Bytes(), &plan); err != nil {
		t.Fatalf("decode profile preview: %v\n%s", err, profileOut.String())
	}
	wantRefs := []string{first.ID, independent.ID}
	if got := plannedRefs(plan.Tasks); !slices.Equal(got, wantRefs) {
		t.Fatalf("profile wave = %v, want compatible width-two wave %v", got, wantRefs)
	}
	if len(plan.Policy.Routing.AllowedHarnesses) != 1 {
		t.Fatalf("profile wave widened single-harness policy: %v", plan.Policy.Routing.AllowedHarnesses)
	}
	harness := plan.Policy.Routing.AllowedHarnesses[0]

	agentsBefore, err := store.ListAgents(w)
	if err != nil {
		t.Fatal(err)
	}
	worktreesBefore, err := gitx.ListWorktrees(w.Root)
	if err != nil {
		t.Fatal(err)
	}
	loopOut := &bytes.Buffer{}
	loopCtx := &clikit.Ctx{Stdout: loopOut, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdLoop(loopCtx, []string{"--project", "p", "--impl-role", "wave-fixer", "--review-role", "wave-reviewer", "--harness", "codex", "--width", "2", "--max-cycles", "1", "--no-pr", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	output := loopOut.String()
	for _, ref := range wantRefs {
		spawnLine := lineContaining(output, "would run: dacli spawn --task "+ref+" ")
		if spawnLine == "" {
			t.Fatalf("loop preview omitted selected task %s:\n%s", ref, output)
		}
		if !strings.Contains(spawnLine, " --harness "+harness) {
			t.Fatalf("loop preview crossed profile harness %q for %s: %s", harness, ref, spawnLine)
		}
	}
	if strings.Contains(output, "would run: dacli spawn --task "+rejected.ID+" ") {
		t.Fatalf("loop preview tried to spawn rejected collision %s:\n%s", rejected.ID, output)
	}
	agentsAfter, err := store.ListAgents(w)
	if err != nil {
		t.Fatal(err)
	}
	worktreesAfter, err := gitx.ListWorktrees(w.Root)
	if err != nil {
		t.Fatal(err)
	}
	if len(agentsAfter) != len(agentsBefore) || len(worktreesAfter) != len(worktreesBefore) {
		t.Fatalf("preview created execution state: agents %d -> %d, worktrees %d -> %d", len(agentsBefore), len(agentsAfter), len(worktreesBefore), len(worktreesAfter))
	}
}

func plannedRefs(tasks []PlannedTask) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.Ref)
	}
	return refs
}

func lineContaining(body, needle string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}

// TestBuildSpawnCarriesClaimDerivedFromTask is the 299 acceptance case for
// "the build phase spawns with a claim derived from the task": the loop used
// to spawn every wave with no --claim at all, so the coordination tool's own
// claim-overlap gate (gateClaimOverlap, execution.go) had nothing to
// arbitrate and two agents could run on overlapping trees at once — an
// operator had to do that arbitration by hand. The claim value must be the
// same path-hint extraction routing already uses (Task.PathHints), not an
// ad-hoc derivation, so the loop's own team.CheapestCapable tie-break and its
// --claim agree on what "the task's files" means.
//
// Narrowed since issue #427: the claim is store.ClaimHints, which is PathHints
// filtered to tokens that RESOLVE to a real path. Routing can tolerate a
// spurious hint (one weak tie-break vote); a claim cannot, because it refuses
// every staged file outside it. So the file the task names has to exist here —
// which is also the honest fixture, since a claim on a path the repo does not
// have could never match anything staged anyway.
func TestBuildSpawnCarriesClaimDerivedFromTask(t *testing.T) {
	w := loopEnv(t)
	if err := os.MkdirAll(filepath.Join(w.Root, "internal", "store"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.Root, "internal", "store", "store.go"), []byte("package store\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "p", "Fix internal/store/store.go bug", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{})
	d.cfg.width = 1
	d.runCycle([]*store.Task{task})

	var buildSpawn []string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, d.cfg.implRole) {
			buildSpawn = c
		}
	}
	if buildSpawn == nil {
		t.Fatal("no build spawn recorded")
	}
	claim := argAfter(buildSpawn, "--claim")
	if claim == "" {
		t.Fatalf("build spawn missing --claim, got: %v", buildSpawn)
	}
	wantHints := strings.Join(store.ClaimHints(w.Root, task), ",")
	if claim != wantHints {
		t.Fatalf("--claim must be the task's own ClaimHints, want %q got %q", wantHints, claim)
	}
	if !strings.Contains(claim, "internal/store/store.go") {
		t.Fatalf("expected the claim to carry the path mentioned in the task title, got %q", claim)
	}
}

func TestBuildSpawnInfersTask371ImplementationScope(t *testing.T) {
	w := loopEnv(t)
	for _, path := range []string{
		"docs/RUNTIMES.md",
		"internal/store/runtimefiles.go",
		"internal/features/execution/execution.go",
		"internal/cli/cli.go",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(w.Root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(w.Root, path), []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	task, err := store.CreateTask(w, "a-root", "p", "Add first-class Codex CLI runtime presets and structured results", store.TaskOpts{Accept: []string{
		"runtime add accepts Codex read-write and read-only presets",
		"The Codex adapter consumes JSONL events and records the final message, session identity, exit outcome, and token usage",
		"runtime doctor verifies Codex read-only isolation through a local sandbox helper",
		"docs/RUNTIMES.md documents Codex as shipped support",
	}})
	if err != nil {
		t.Fatal(err)
	}

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{})
	d.cfg.width = 1
	d.runCycle([]*store.Task{task})

	var claim string
	for _, call := range fr.calls {
		if len(call) > 0 && call[0] == "spawn" && contains(call, d.cfg.implRole) {
			claim = argAfter(call, "--claim")
		}
	}
	for _, want := range []string{"docs/RUNTIMES.md", "internal/store", "internal/features/execution", "internal/cli"} {
		if !contains(strings.Split(claim, ","), want) {
			t.Errorf("spawn --claim %q missing %q", claim, want)
		}
	}
	if contains(strings.Split(claim, ","), "internal/features/orchestration") {
		t.Errorf("spawn --claim %q includes an unrelated path", claim)
	}
	claims := strings.Split(claim, ",")
	for _, required := range []string{
		"docs/RUNTIMES.md",
		"internal/store/runtimefiles.go",
		"internal/cli/runtime_test.go",
		"internal/features/execution/execution.go",
		"internal/features/execution/execruntime_test.go",
		"internal/features/execution/runrecord_test.go",
		"internal/features/execution/stream_test.go",
	} {
		if _, _, covered := procmon.PathsOverlap([]string{required}, claims); !covered {
			t.Errorf("spawn --claim %q would refuse task 371 file %q", claim, required)
		}
	}
	if _, _, covered := procmon.PathsOverlap([]string{"internal/features/orchestration/orchestration.go"}, claims); covered {
		t.Errorf("spawn --claim %q would allow an unrelated file", claim)
	}
}

// TestBuildSpawnOmitsClaimWhenTaskNamesNoPath proves the flag is only added
// when there is something to claim: splitClaims on the spawn side treats an
// empty --claim value as no claim at all, so appending "--claim" with
// nothing after it would just be noise on every task whose text names no
// file.
func TestBuildSpawnOmitsClaimWhenTaskNamesNoPath(t *testing.T) {
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Improve onboarding copy", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(task.PathHints()) != 0 {
		t.Fatalf("test setup: expected a path-free task title, got hints %v", task.PathHints())
	}

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{})
	d.cfg.width = 1
	d.runCycle([]*store.Task{task})

	var buildSpawn []string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, d.cfg.implRole) {
			buildSpawn = c
		}
	}
	if buildSpawn == nil {
		t.Fatal("no build spawn recorded")
	}
	if contains(buildSpawn, "--claim") {
		t.Fatalf("expected no --claim flag for a task with no path hints, got: %v", buildSpawn)
	}
}

// TestClaimConflictReschedulesTaskNextCycleRatherThanFailingWave is the 299
// acceptance case for "a claim conflict schedules the task into the next
// cycle rather than failing the wave": a spawn refused with a path-claim
// conflict must not abort the rest of the wave, and the conflicting task
// must stay open (never pendingAccept, never force-closed) so the very next
// cycle's ready frontier re-offers it — the same "leave it for next cycle"
// contract a taint/budget refusal already gets, just for the claim gate.
func TestClaimConflictReschedulesTaskNextCycleRatherThanFailingWave(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt")
	conflicted, err := store.CreateTask(w, "a-root", "p", "Task whose claim conflicts with a live agent", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	sibling, err := store.CreateTask(w, "a-root", "p", "Task that builds fine", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	conflictRef := conflicted.ID

	r := &spawnOutcomeRunner{w: w, refusedRef: conflictRef}
	// spawnOutcomeRunner's generic refusal message stands in for gateClaimOverlap's
	// real one; what this test asserts is the loop's REACTION to a spawn error —
	// identical regardless of which gate produced it — so replacing the error
	// text is enough to exercise the same code path a live claim conflict would.
	r.refusalMsg = "path-claim conflict: live agent a-x already claims \"internal/store\" and you claim \"internal/store\" — narrow your scope, or `dacli wait 01ABC` first"

	d := newDriver(w, r, &Governor{})
	d.cfg.width = 2

	d.runCycle([]*store.Task{conflicted, sibling})

	// The sibling must still have built despite the conflict earlier in the
	// wave — a claim conflict on one task must not fail the whole wave.
	pending := map[int]bool{}
	for _, p := range d.pendingAccept {
		pending[p.Seq] = true
	}
	if !pending[sibling.Seq] {
		t.Fatalf("sibling task %03d must still build despite the earlier claim conflict, pendingAccept: %v", sibling.Seq, d.pendingAccept)
	}
	if pending[conflicted.Seq] {
		t.Fatalf("conflicted task %03d must never be tracked as pending accept: %v", conflicted.Seq, d.pendingAccept)
	}

	// The conflicted task must remain open — ready for the NEXT cycle to
	// re-offer it, not silently dropped or force-closed.
	open, err := store.ListTasks(w, "p", model.StatusOpen)
	if err != nil {
		t.Fatal(err)
	}
	foundOpen := false
	for _, tk := range open {
		if tk.Seq == conflicted.Seq {
			foundOpen = true
		}
	}
	if !foundOpen {
		t.Fatalf("conflicted task %03d must stay open for the next cycle to re-pick", conflicted.Seq)
	}

	ready := excludePending(open, d.pendingAccept)
	foundReady := false
	for _, tk := range ready {
		if tk.Seq == conflicted.Seq {
			foundReady = true
		}
	}
	if !foundReady {
		t.Fatalf("conflicted task %03d must be part of the next cycle's ready frontier", conflicted.Seq)
	}
}
