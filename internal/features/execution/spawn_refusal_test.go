package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const testProject = "proj"

// newExecWS builds a workspace with one project, acting as the root (rw)
// identity. DACLI_AGENT is cleared so an inherited token from a dacli-spawned
// test run cannot change which identity these tests act as.
func newExecWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	t.Setenv(agentid.EnvVar, "")
	w, err := workspace.Init(t.TempDir(), "exec-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "Test project", testProject, "Ship it.", ""); err != nil {
		t.Fatal(err)
	}
	return w
}

func mustTask(t *testing.T, w *workspace.Workspace, title string, opts store.TaskOpts) *store.Task {
	t.Helper()
	task, err := store.CreateTask(w, agentid.RootID, testProject, title, opts)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func setPhase(t *testing.T, w *workspace.Workspace, phase string, allows []string) {
	t.Helper()
	p, err := store.LoadProject(w, testProject)
	if err != nil {
		t.Fatal(err)
	}
	p.Doc.Front.Set("phase", phase)
	p.Doc.Front.SetList("phase_allows", allows)
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
}

func clearPhase(t *testing.T, w *workspace.Workspace) {
	t.Helper()
	p, err := store.LoadProject(w, testProject)
	if err != nil {
		t.Fatal(err)
	}
	p.Doc.Front.Delete("phase")
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
}

// fakeBinary writes an executable no-op script and returns its absolute path,
// so a runtime can clear the exec.LookPath check without any real agent CLI
// being installed. Nothing in these tests ever runs it: every case asserts a
// refusal that happens BEFORE the process would start.
func fakeBinary(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fake-agent")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func mustRuntime(t *testing.T, w *workspace.Workspace, rt store.Runtime) {
	t.Helper()
	if err := store.CreateRuntime(w, agentid.RootID, rt, ""); err != nil {
		t.Fatal(err)
	}
}

func mustRole(t *testing.T, w *workspace.Workspace, r team.Role) {
	t.Helper()
	if err := store.CreateRole(w, agentid.RootID, r); err != nil {
		t.Fatal(err)
	}
}

// countAgents / countRuns measure the side effects a refusal must NOT have.
func countAgents(t *testing.T, w *workspace.Workspace) int {
	t.Helper()
	entries, _ := os.ReadDir(w.AgentsDir())
	return len(entries)
}

func countRuns(t *testing.T, w *workspace.Workspace) int {
	t.Helper()
	entries, _ := os.ReadDir(w.RunsDir())
	return len(entries)
}

// TestSpawnRefusals enumerates every decision `dacli spawn` makes BEFORE it
// launches a process. Each is a pure policy decision, so each is asserted for
// its exit code (the 2/3/4 contract a supervisor branches on), for a message
// that names the actual reason, and — the load-bearing part — for having minted
// NO child identity and written NO run record. A refusal that leaks a run dir
// or an agent file has already had the side effect it was supposed to prevent.
func TestSpawnRefusalsHappenBeforeAnyProcessStarts(t *testing.T) {
	bin := fakeBinary(t)

	cases := []struct {
		name     string
		setup    func(t *testing.T, w *workspace.Workspace) []string // returns spawn args
		wantExit int
		wantMsg  string
	}{
		{
			name: "no --task is a usage error",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				return nil
			},
			wantExit: 2,
			wantMsg:  "usage: dacli spawn",
		},
		{
			name: "an unknown flag is a usage error, never silently dropped",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				return []string{"--task", "001", "--runtimee", "x"}
			},
			wantExit: 2,
			wantMsg:  "unknown flag",
		},
		{
			name: "an unknown task is not found",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				return []string{"--task", "999"}
			},
			wantExit: 4,
		},
		{
			name: "no runtime named anywhere is a usage error",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				return []string{"--task", "001"}
			},
			wantExit: 2,
			wantMsg:  "no runtime",
		},
		{
			name: "a role at its WIP limit refuses the spawn",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				mustRole(t, w, team.Role{Name: "junior", Runtime: "rt", WIP: 1})
				// One live agent already holds the role's only slot.
				if _, _, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, "junior", model.GrantRO); err != nil {
					t.Fatal(err)
				}
				return []string{"--task", "001", "--role", "junior"}
			},
			wantExit: 3,
			wantMsg:  "WIP limit (1/1)",
		},
		{
			name: "a seniority-capped role refuses an unestimated task",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "unsized task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				mustRole(t, w, team.Role{Name: "junior", Runtime: "rt", MaxPoints: 3})
				return []string{"--task", "001", "--role", "junior"}
			},
			wantExit: 3,
			wantMsg:  "takes only estimated tasks",
		},
		{
			name: "a seniority-capped role refuses a task above its cap",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "big task", store.TaskOpts{Estimate: "5,10,20"})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				mustRole(t, w, team.Role{Name: "junior", Runtime: "rt", MaxPoints: 3})
				return []string{"--task", "001", "--role", "junior"}
			},
			wantExit: 3,
			wantMsg:  "above role junior's cap",
		},
		{
			name: "a gated phase refuses a role whose kind has no work there",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				mustRole(t, w, team.Role{Name: "impl", Runtime: "rt", Kind: "implementer"})
				setPhase(t, w, "discovery", []string{"researcher"})
				return []string{"--task", "001", "--role", "impl"}
			},
			wantExit: 3,
			wantMsg:  "is in the discovery phase",
		},
		{
			name: "an unknown runtime is not found",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				return []string{"--task", "001", "--runtime", "nope"}
			},
			wantExit: 4,
		},
		{
			name: "a runtime whose binary is not installed is refused, not attempted",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: "dacli-no-such-binary-xyz", Mode: "stdin"})
				return []string{"--task", "001", "--runtime", "rt"}
			},
			wantExit: 1,
			wantMsg:  "not on PATH",
		},
		{
			name: "a read-only grant on a runtime that cannot enforce it is refused",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin"}) // no sandbox_ro
				return []string{"--task", "001", "--runtime", "rt", "--grant", "ro"}
			},
			wantExit: 3,
			wantMsg:  "cannot enforce read-only",
		},
		{
			name: "an rw grant on a runtime with no write tool is refused, not launched",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				// The junior/cc shape: the only allowlist is the read-only
				// sandbox, so an rw child has no Edit/Write and cannot work.
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "arg", Flag: "-p",
					SandboxRO: []string{"--allowedTools", "Read,Grep,Glob,LS"}})
				return []string{"--task", "001", "--runtime", "rt", "--grant", "rw"}
			},
			wantExit: 3,
			wantMsg:  "no write tool",
		},
		{
			name: "a tainted brief blocks the spawn rather than feeding it to a child",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				task := mustTask(t, w, "some task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				if _, err := eventlog.Append(w, agentid.RootID, model.EventFinding, task.ID,
					"external:drive-by-issue", "Suggested fix from an internet stranger"); err != nil {
					t.Fatal(err)
				}
				return []string{"--task", "001", "--runtime", "rt"}
			},
			wantExit: 3,
			wantMsg:  "blast radius of external:drive-by-issue",
		},
		{
			name: "a path-claim conflict with a live agent is refused",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				writeLiveProcRecord(t, w, []string{"internal/store"})
				// internal/store/roles.go is INSIDE the live agent's claim.
				return []string{"--task", "001", "--runtime", "rt", "--claim", "internal/store/roles.go"}
			},
			wantExit: 3,
			wantMsg:  "path-claim conflict",
		},
		{
			name: "--max-tokens takes a positive integer",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				return []string{"--task", "001", "--runtime", "rt", "--max-tokens", "0"}
			},
			wantExit: 2,
			wantMsg:  "positive integer",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newExecWS(t)
			args := tc.setup(t, w)
			agentsBefore, runsBefore := countAgents(t, w), countRuns(t, w)

			ctx, out, errb := newCtx(w.Root)
			err := cmdSpawn(ctx, args)
			if code := clikit.ExitCode(err); code != tc.wantExit {
				t.Fatalf("cmdSpawn(%v) exit %d, want %d (err %v)\nstdout: %s\nstderr: %s",
					args, code, tc.wantExit, err, out, errb)
			}
			if err == nil {
				t.Fatal("expected a refusal, got success")
			}
			if tc.wantMsg != "" && !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("refusal %q does not contain %q", err, tc.wantMsg)
			}
			// The point of refusing early: nothing was minted and nothing ran.
			if got := countAgents(t, w); got != agentsBefore {
				t.Errorf("a refused spawn minted %d child identity/ies", got-agentsBefore)
			}
			if got := countRuns(t, w); got != runsBefore {
				t.Errorf("a refused spawn wrote %d run record(s)", got-runsBefore)
			}
		})
	}
}

// writeLiveProcRecord fabricates a run whose proc.txt names THIS test process —
// a genuinely live pid with a matching start time, so liveAgents sees a live
// agent without anything being spawned. Nothing is ever signalled.
func writeLiveProcRecord(t *testing.T, w *workspace.Workspace, claims []string) procmon.Record {
	t.Helper()
	pid := os.Getpid()
	start, ok := procmon.ProcStart(pid)
	if !ok {
		t.Skip("ps cannot read this process's start time")
	}
	runID := ulid.New()
	dir := w.RunDir(runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{
		RunID: runID, Child: "a-live", Task: "t-live", Role: "junior", Runtime: "rt",
		PID: pid, PGID: pid, PIDStart: start, Started: time.Now(), Claims: claims,
	}
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

// A DISJOINT claim must NOT be refused — the conflict check has to be a real
// discriminator, not a blanket "any live agent blocks you". Without this the
// conflict test above would pass on a function that always refuses.
func TestSpawnAllowsDisjointClaims(t *testing.T) {
	w := newExecWS(t)
	mustTask(t, w, "some task", store.TaskOpts{})
	mustRuntime(t, w, store.Runtime{Name: "rt", Binary: fakeBinary(t), Mode: "stdin"})
	writeLiveProcRecord(t, w, []string{"internal/store"})

	ctx, _, _ := newCtx(w.Root)
	// grant ro on a runtime with no sandbox refuses at the NEXT gate — which
	// proves the claim check let this through rather than that nothing ran.
	err := cmdSpawn(ctx, []string{"--task", "001", "--runtime", "rt", "--claim", "internal/storefront"})
	if err == nil {
		t.Fatal("expected the sandbox refusal that follows the claim check")
	}
	if strings.Contains(err.Error(), "path-claim conflict") {
		t.Errorf("internal/storefront must not conflict with internal/store: %v", err)
	}
	if !strings.Contains(err.Error(), "cannot enforce read-only") {
		t.Errorf("expected to reach the sandbox gate, got %v", err)
	}
}

// The taint gate is an override, not a wall: --force is the loud, explicit
// acceptance of the risk, so it must get PAST the taint refusal (and land on
// the next gate) rather than being refused anyway.
func TestTaintGateIsOverriddenByForce(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "some task", store.TaskOpts{})
	mustRuntime(t, w, store.Runtime{Name: "rt", Binary: fakeBinary(t), Mode: "stdin"})
	if _, err := eventlog.Append(w, agentid.RootID, model.EventFinding, task.ID,
		"external:drive-by-issue", "content from outside the tree"); err != nil {
		t.Fatal(err)
	}

	ctx, _, _ := newCtx(w.Root)
	err := cmdSpawn(ctx, []string{"--task", "001", "--runtime", "rt", "--force"})
	if err == nil {
		t.Fatal("expected the sandbox refusal that follows the taint gate")
	}
	if strings.Contains(err.Error(), "blast radius") {
		t.Errorf("--force must override the taint gate; still refused: %v", err)
	}
	if !strings.Contains(err.Error(), "cannot enforce read-only") {
		t.Errorf("expected to reach the sandbox gate, got %v", err)
	}
}

// `spawn --advise` must PREVIEW and stop, never launch (dacli 232). The word
// "advise" means the same thing on `spawn` as it does on `loop`: price this
// launch from the log, print it, and act on nothing. The load-bearing assertion
// is the side-effect count — a preview that mints an identity or writes a run
// record has already spent the very run the operator asked only to price. The
// runtime here CAN enforce read-only and has an installed binary, so BEFORE this
// change the spawn would have sailed past every gate and actually run.
func TestSpawnAdvisePreviewsWithoutSpawning(t *testing.T) {
	w := newExecWS(t)
	mustTask(t, w, "estimated task", store.TaskOpts{Estimate: "1,2,3"})
	mustRuntime(t, w, store.Runtime{Name: "rt", Binary: fakeBinary(t), Mode: "stdin", SandboxRO: []string{"--ro"}})

	agentsBefore, runsBefore := countAgents(t, w), countRuns(t, w)

	ctx, out, _ := newCtx(w.Root)
	err := cmdSpawn(ctx, []string{"--task", "001", "--runtime", "rt", "--advise"})
	if err != nil {
		t.Fatalf("spawn --advise should preview and exit 0, got %v", err)
	}
	if got := countAgents(t, w); got != agentsBefore {
		t.Errorf("a preview minted %d child identity/ies — --advise must not spawn", got-agentsBefore)
	}
	if got := countRuns(t, w); got != runsBefore {
		t.Errorf("a preview wrote %d run record(s) — --advise must not spawn", got-runsBefore)
	}
	got := out.String()
	if !strings.Contains(got, "advise ·") {
		t.Errorf("advise output missing the sizing readout, got: %s", got)
	}
	if !strings.Contains(got, "no agent spawned") {
		t.Errorf("a preview must say plainly that nothing launched, got: %s", got)
	}
}

// The --max-tokens gate must only enforce what it can measure honestly. With
// no token history for the band (or no estimate on the task) there is nothing
// to compare against, so it NOTES that it is not enforcing and proceeds —
// silently refusing on an unmeasurable band would block every first spawn, and
// silently passing without saying so would make the flag look effective when
// it is inert.
func TestMaxTokensIsNotEnforcedWithoutMeasuredCost(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "estimated task", store.TaskOpts{Estimate: "1,2,3"})
	mustRuntime(t, w, store.Runtime{Name: "rt", Binary: fakeBinary(t), Mode: "stdin"})

	band := store.Band{Role: "-", Model: "-", Runtime: "rt"}
	if _, n, ok := bandTokenBudget(w, task, band); ok || n != 0 {
		t.Errorf("bandTokenBudget with no run history = (n %d, ok %v); want (0, false)", n, ok)
	}

	ctx, _, errb := newCtx(w.Root)
	err := cmdSpawn(ctx, []string{"--task", "001", "--runtime", "rt", "--max-tokens", "100"})
	if err == nil || !strings.Contains(err.Error(), "cannot enforce read-only") {
		t.Fatalf("expected to reach the sandbox gate, got %v", err)
	}
	if !strings.Contains(errb.String(), "not enforced") {
		t.Errorf("an inert --max-tokens must say so; stderr was %q", errb)
	}
}

// An UNESTIMATED task has no Te to multiply the band ratio by, so there is
// likewise nothing to enforce — reported as not-ok rather than as a zero
// budget, which would refuse every spawn.
func TestBandTokenBudgetNeedsAnEstimate(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "unsized task", store.TaskOpts{})
	if expected, _, ok := bandTokenBudget(w, task, store.Band{Runtime: "rt"}); ok || expected != 0 {
		t.Errorf("bandTokenBudget on an unestimated task = (%v, ok %v); want (0, false)", expected, ok)
	}
}

// externalRadius must distinguish three states, because --advise prints a
// different line for each and the gate only fires on one: not in the radius
// with nothing external recorded, not in the radius with something external
// recorded elsewhere, and in the radius.
func TestExternalRadiusDistinguishesCleanFromNotRecorded(t *testing.T) {
	w := newExecWS(t)
	tainted := mustTask(t, w, "tainted task", store.TaskOpts{})

	if _, inRadius, hasExternal := externalRadius(w, tainted); inRadius || hasExternal {
		t.Errorf("with nothing recorded: inRadius=%v hasExternal=%v, want false/false", inRadius, hasExternal)
	}

	if _, err := eventlog.Append(w, agentid.RootID, model.EventFinding, tainted.ID,
		"external:some-user", "drive-by"); err != nil {
		t.Fatal(err)
	}
	origins, inRadius, hasExternal := externalRadius(w, tainted)
	if !inRadius || !hasExternal {
		t.Fatalf("tainted task: inRadius=%v hasExternal=%v, want true/true", inRadius, hasExternal)
	}
	if len(origins) != 1 || origins[0] != "external:some-user" {
		t.Errorf("origins = %v, want [external:some-user]", origins)
	}

	// A DIFFERENT task, with an external artifact recorded but not reaching it.
	clean := mustTask(t, w, "clean task", store.TaskOpts{})
	_, inRadius, hasExternal = externalRadius(w, clean)
	if inRadius {
		t.Error("an unattached external artifact must not taint every task")
	}
	if !hasExternal {
		t.Error("hasExternal must stay true — 'clean' and 'nothing recorded' are different answers")
	}
}
