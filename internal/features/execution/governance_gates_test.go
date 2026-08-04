package execution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// spawningCommands is the closed set of commands that put a brief in front of a
// runtime and start a child. `supervise` runs the SAME brief through the SAME
// runtime as `spawn`, once per turn, so every governance gate spawn enforces
// must bind supervise identically — otherwise the whole gate set is opt-out by
// typing a different verb (dacli 239).
//
// extra holds the GENUINE per-command differences only (supervise's turn cap,
// pinned to 1 so a regression cannot sit in a three-turn loop). Nothing
// gate-relevant belongs here: a flag that changes a gate's verdict would let a
// command look compliant on a technicality.
var spawningCommands = []struct {
	name  string
	run   func(*clikit.Ctx, []string) error
	extra []string
}{
	{"spawn", cmdSpawn, nil},
	{"supervise", cmdSupervise, []string{"--max-turns", "1"}},
}

// TestEveryGateRefusesInEverySpawningCommand is the anti-drift test for the
// shared pre-launch path. Each row is ONE governance gate, expressed in
// command-agnostic args, and is run against EVERY entry of spawningCommands.
//
// Adding a gate to the system therefore means adding exactly one row here, and
// that row immediately demands the gate of both commands. This is the property
// that failed in dacli 239: cmdSupervise had copied cmdSpawn's prologue and
// dropped the WIP, taint, --max-tokens and --claim gates, so a supervised run
// bypassed all four.
//
// Every gate is asserted at exit 3 — the refusal contract a supervisor branches
// on — and, load-bearingly, at ZERO side effects: no child identity minted, no
// run record written. A gate that fires after the process starts has already
// permitted what it was meant to prevent.
func TestEveryGateRefusesInEverySpawningCommand(t *testing.T) {
	bin := fakeBinary(t)

	gates := []struct {
		gate    string
		setup   func(t *testing.T, w *workspace.Workspace) []string // command-agnostic args
		wantMsg string
	}{
		{
			gate: "role WIP limit",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				mustRole(t, w, team.Role{Name: "junior", Runtime: "rt", WIP: 1})
				// A live agent already holds the role's only slot.
				if _, _, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, "junior", model.GrantRO); err != nil {
					t.Fatal(err)
				}
				return []string{"--task", "001", "--role", "junior"}
			},
			wantMsg: "WIP limit (1/1)",
		},
		{
			gate: "tainted brief (blast radius)",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				task := mustTask(t, w, "some task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				if _, err := eventlog.Append(w, agentid.RootID, model.EventFinding, task.ID,
					"external:drive-by-issue", "Suggested fix from an internet stranger"); err != nil {
					t.Fatal(err)
				}
				return []string{"--task", "001", "--runtime", "rt"}
			},
			wantMsg: "blast radius of external:drive-by-issue",
		},
		{
			gate: "--max-tokens budget",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				// Te 2.0, so the band's 500 output-tokens/point projects ~1000.
				mustTask(t, w, "estimated task", store.TaskOpts{Estimate: "1,2,3"})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				seedTokenHistory(t, w, "rt", 10, 1000)
				return []string{"--task", "001", "--runtime", "rt", "--max-tokens", "100"}
			},
			wantMsg: "exceeds --max-tokens 100",
		},
		{
			gate: "--claim overlaps a live agent",
			setup: func(t *testing.T, w *workspace.Workspace) []string {
				mustTask(t, w, "some task", store.TaskOpts{})
				mustRuntime(t, w, store.Runtime{Name: "rt", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})
				writeLiveProcRecord(t, w, []string{"internal/store"})
				// internal/store/roles.go is INSIDE the live agent's claim.
				return []string{"--task", "001", "--runtime", "rt", "--claim", "internal/store/roles.go"}
			},
			wantMsg: "path-claim conflict",
		},
	}

	for _, g := range gates {
		for _, c := range spawningCommands {
			t.Run(g.gate+"/"+c.name, func(t *testing.T) {
				w := newExecWS(t)
				args := append(g.setup(t, w), c.extra...)
				agentsBefore, runsBefore := countAgents(t, w), countRuns(t, w)

				ctx, out, errb := newCtx(w.Root)
				err := c.run(ctx, args)
				if code := clikit.ExitCode(err); code != 3 {
					t.Fatalf("dacli %s %v: exit %d, want 3 (gate %q not enforced)\nerr: %v\nstdout: %s\nstderr: %s",
						c.name, args, code, g.gate, err, out, errb)
				}
				if !strings.Contains(err.Error(), g.wantMsg) {
					t.Errorf("dacli %s refusal %q does not name the reason %q", c.name, err, g.wantMsg)
				}
				if got := countAgents(t, w); got != agentsBefore {
					t.Errorf("dacli %s: a refused launch minted %d child identity/ies", c.name, got-agentsBefore)
				}
				if got := countRuns(t, w); got != runsBefore {
					t.Errorf("dacli %s: a refused launch wrote %d run record(s)", c.name, got-runsBefore)
				}
			})
		}
	}
}

// Every spawning command must accept every flag the shared prologue reads.
// A command that rejects --claim or --max-tokens as unknown cannot be gated by
// them at all (and one that rejects --force cannot be un-gated deliberately),
// which is the second half of the 239 hole: supervise's flag set had no --claim
// and no --max-tokens, so those gates were unreachable rather than merely
// unenforced.
func TestEverySpawningCommandAcceptsTheSharedLaunchFlags(t *testing.T) {
	for _, c := range spawningCommands {
		for _, flag := range launchFlags {
			t.Run(c.name+"/--"+flag, func(t *testing.T) {
				w := newExecWS(t)
				ctx, _, _ := newCtx(w.Root)
				// No task exists: the command gets as far as flag validation and
				// then fails on the task lookup. An "unknown flag" here means the
				// flag is not part of this command's surface at all.
				args := append([]string{"--task", "001", "--" + flag, "x"}, c.extra...)
				err := c.run(ctx, args)
				if err != nil && strings.Contains(err.Error(), "unknown flag") {
					t.Errorf("dacli %s rejects --%s, a flag the shared prologue reads: %v", c.name, flag, err)
				}
			})
		}
	}
}

// seedTokenHistory fabricates n completed, token-bearing calibration samples in
// band -/-/<runtime>: each is a done task with a three-point estimate (Te = 2.0)
// and a two-hour claim→completion span, joined to a run record reporting tokens
// output tokens. n ≥ 10 is what lifts the --max-tokens gate out of its
// PROVISIONAL warn-only regime and into a hard refusal.
func seedTokenHistory(t *testing.T, w *workspace.Workspace, runtime string, n, tokens int) {
	t.Helper()
	for i := 0; i < n; i++ {
		task := mustTask(t, w, fmt.Sprintf("history %d", i), store.TaskOpts{Estimate: "1,2,3"})
		task.Doc.SetSection("Log",
			"- 2026-07-20T09:00:00Z claimed by a-root\n- 2026-07-20T11:00:00Z completed by a-root\n")
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
		if err := store.MoveTask(w, task, model.StatusDone); err != nil {
			t.Fatal(err)
		}
		dir := w.RunDir(ulid.New())
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		inv := fmt.Sprintf("task: %s\nrole: -\nmodel: -\nruntime: %s\n", task.ID, runtime)
		if err := os.WriteFile(filepath.Join(dir, "invocation.txt"), []byte(inv), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "usage.txt"),
			[]byte(fmt.Sprintf("output_tokens: %d\n", tokens)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
