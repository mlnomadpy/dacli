package orchestration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
)

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
	conflictRef := fmt.Sprintf("%03d", conflicted.Seq)

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
