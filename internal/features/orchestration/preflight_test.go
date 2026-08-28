package orchestration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
)

type recordingPreflightRunner struct {
	fakeRunner
	failRole string
}

func (r *recordingPreflightRunner) runPreflight(label string, args ...string) (string, error) {
	r.calls = append(r.calls, args)
	if argAfter(args, "--role") == r.failRole {
		return "runtime contract incompatible", clikit.Refusedf("runtime contract incompatible")
	}
	return "compatible", nil
}

func TestExplicitUnderCapacityRoleRefusesBeforeAnyWorkerStarts(t *testing.T) {
	w := loopEnv(t)
	for _, role := range []team.Role{
		{Name: "junior", Kind: "implementer", Grant: "rw", MaxPoints: 3, Summary: "small fixes"},
		{Name: "reviewer", Kind: "reviewer", Grant: "ro", Summary: "reviews"},
	} {
		if err := store.CreateRole(w, "a-root", role); err != nil {
			t.Fatal(err)
		}
	}
	task, err := store.CreateTask(w, "a-root", "p", "Implement large change", store.TaskOpts{Accept: []string{"observable"}, Estimate: "8,10,12"})
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingPreflightRunner{}
	d := newDriver(w, runner, &Governor{MaxCycles: 1, NoProgressHalt: 3})
	d.cfg.implRole, d.cfg.implRoleExplicit, d.cfg.reviewRole = "junior", true, "reviewer"
	err = d.loop()
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "above role junior's cap") {
		t.Fatalf("loop error = %v (exit %d), want capacity refusal", err, clikit.ExitCode(err))
	}
	for _, call := range runner.calls {
		if len(call) > 0 && call[0] == "spawn" {
			t.Fatalf("capacity refusal happened after worker spawn: %v", call)
		}
	}
	raw, err := os.ReadFile(cyclePreflightFile(w, "p"))
	if err != nil {
		t.Fatalf("missing durable preflight result: %v", err)
	}
	var result cyclePreflightResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("preflight result is not JSON: %v", err)
	}
	if result.SchemaVersion != 1 || result.Verdict != "refuse" || result.Phases[0].Task != task.ID || result.Phases[0].Capacity.Delta != -7 {
		t.Fatalf("preflight result lost versioned capacity evidence: %+v", result)
	}
}

func TestReviewerPreflightRefusesBeforeImplementationSpawn(t *testing.T) {
	w := loopEnv(t)
	for _, role := range []team.Role{
		{Name: "builder", Kind: "implementer", Grant: "rw", Summary: "builds"},
		{Name: "broken-reviewer", Kind: "reviewer", Grant: "ro", Summary: "reviews"},
	} {
		if err := store.CreateRole(w, "a-root", role); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateTask(w, "a-root", "p", "Implement reviewed change", store.TaskOpts{Accept: []string{"observable"}, Estimate: "2,2,2"}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingPreflightRunner{failRole: "broken-reviewer"}
	d := newDriver(w, runner, &Governor{MaxCycles: 3, NoProgressHalt: 3})
	d.cfg.implRole, d.cfg.implRoleExplicit, d.cfg.reviewRole = "builder", true, "broken-reviewer"
	err := d.loop()
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("loop error = %v (exit %d), want permanent reviewer refusal", err, clikit.ExitCode(err))
	}
	if len(runner.calls) != 1 || runner.calls[0][0] != "preflight" || argAfter(runner.calls[0], "--role") != "broken-reviewer" {
		t.Fatalf("reviewer must be the only command reached before refusal: %v", runner.calls)
	}
}

func TestCyclePreflightPreservesMultiSliceClaimsDespiteIncidentalProse(t *testing.T) {
	w := loopEnv(t)
	for _, path := range []string{
		"internal/features/ship/ship.go",
		"internal/features/vcs/vcs.go",
		"internal/features/acceptance/acceptance.go",
		"internal/features/execution/execution.go",
	} {
		full := filepath.Join(w.Root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.CreateRole(w, "a-root", team.Role{Name: "lifecycle", Kind: "implementer", Grant: "rw", Summary: "lifecycle work"}); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "p", "Fix product lifecycle", store.TaskOpts{
		Estimate: "3,3,3",
		Context:  "The report incidentally mentions internal/features/execution, but the lifecycle crosses slices.",
		Accept: []string{
			"internal/features/ship/ship.go handles landing",
			"internal/features/vcs/vcs.go preserves the branch",
			"internal/features/acceptance/acceptance.go verifies closure",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	d := newDriver(w, &fakeRunner{}, &Governor{})
	d.cfg.implRole, d.cfg.implRoleExplicit = "lifecycle", true
	if err := d.preflightCycle([]*store.Task{task}); err != nil {
		t.Fatal(err)
	}
	claims := d.ctx.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{"internal/features/ship/ship.go", "internal/features/vcs/vcs.go", "internal/features/acceptance/acceptance.go"} {
		if !strings.Contains(claims, want) {
			t.Errorf("versioned preflight collapsed multi-slice claim; missing %s in %s", want, claims)
		}
	}
}
