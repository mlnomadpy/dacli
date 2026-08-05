package agentstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func testWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	t.Setenv(agentid.EnvVar, "")
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "Core", "core", "goal", "build"); err != nil {
		t.Fatalf("project: %v", err)
	}
	return w
}

// liveRecord fabricates a run whose transcript.log holds body (mtime backdated
// by age; 0 leaves it fresh) and whose proc.txt names runtime. RunID is fixed
// per test so callers can find the run dir deterministically.
func liveRecord(t *testing.T, w *workspace.Workspace, runID, runtime, task, body string, age time.Duration) procmon.Record {
	t.Helper()
	dir := w.RunDir(runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("rundir: %v", err)
	}
	path := filepath.Join(dir, "transcript.log")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("transcript: %v", err)
	}
	if age > 0 {
		mt := time.Now().Add(-age)
		if err := os.Chtimes(path, mt, mt); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}
	return procmon.Record{RunID: runID, Task: task, Runtime: runtime, Started: time.Now().Add(-age)}
}

// TestDeriveCoversEveryState is the invariant test for the state machine
// `dacli agents` and the dashboard both call into: every one of the six
// states Derive can return, exercised through the same function signature
// both callers use, so a future caller cannot silently reimplement a subset.
func TestDeriveCoversEveryState(t *testing.T) {
	w := testWS(t)
	if err := store.CreateRuntime(w, agentid.RootID, store.Runtime{Name: "streamy", Binary: "streamy", Mode: "stdin", UsageFormat: "stream-json"}, ""); err != nil {
		t.Fatalf("runtime: %v", err)
	}
	if err := store.CreateRuntime(w, agentid.RootID, store.Runtime{Name: "texty", Binary: "texty", Mode: "stdin"}, ""); err != nil {
		t.Fatalf("runtime: %v", err)
	}

	cases := []struct {
		name    string
		runtime string
		body    string
		age     time.Duration
		task    string
		want    string
	}{
		{"assistant prose, fresh", "streamy", "Looking at the approach.\n", 0, "", Thinking},
		{"tool marker, fresh", "streamy", "Looking at the file.\n[tool: Read]\n", 0, "", Acting},
		{"raw stream-json tool_use", "streamy", `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash"}]}}` + "\n", 0, "", Acting},
		{"empty transcript, fresh, stream runtime", "streamy", "", 0, "", Waiting},
		{"empty transcript, frozen, stream runtime", "streamy", "", 5 * time.Minute, "", Stalled},
		{"prose that stopped moving", "streamy", "Still reasoning about it.\n", 5 * time.Minute, "", Stalled},
		{"empty transcript, fresh, text runtime", "texty", "", 0, "", Waiting},
		{"empty transcript, frozen, text runtime", "texty", "", 5 * time.Minute, "", Silent},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runID := runIDFor(i)
			rec := liveRecord(t, w, runID, c.runtime, c.task, c.body, c.age)
			if got := Derive(w, rec, nil); got != c.want {
				t.Errorf("Derive = %q, want %q", got, c.want)
			}
		})
	}
}

func runIDFor(i int) string {
	return "01RUNAGENTSTATE" + string(rune('A'+i)) + "000000000"
}

// TestDeriveReportsBlockedRegardlessOfTranscript proves blocked overrides
// every transcript-derived state — an agent mid-tool-call whose task has an
// outstanding `dacli ask` is waiting on a human, not "acting".
func TestDeriveReportsBlockedRegardlessOfTranscript(t *testing.T) {
	w := testWS(t)
	task, err := store.CreateTask(w, agentid.RootID, "core", "Needs an answer", store.TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	if err := store.MoveTask(w, task, model.StatusBlocked); err != nil {
		t.Fatalf("move: %v", err)
	}
	tasks, err := store.BuildTaskIndex(w)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	rec := liveRecord(t, w, "01RUNBLOCKEDAGENT00000000", "streamy", task.Slug, "Looking at the file.\n[tool: Read]\n", 0)

	if got := Derive(w, rec, tasks); got != Blocked {
		t.Errorf("Derive = %q, want blocked (task has an outstanding ask)", got)
	}

	// An open (unblocked) task falls through to the transcript-derived state.
	if err := store.MoveTask(w, task, model.StatusOpen); err != nil {
		t.Fatalf("move: %v", err)
	}
	tasks, err = store.BuildTaskIndex(w)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if got := Derive(w, rec, tasks); got != Acting {
		t.Errorf("Derive = %q, want acting once the task is unblocked", got)
	}
}

// TestDeriveDegradesToNeverBlockedWithoutAnIndex documents the nil-index
// fallback: a caller that skips building a TaskIndex (or whose build failed)
// gets transcript-only states, never a false "blocked".
func TestDeriveDegradesToNeverBlockedWithoutAnIndex(t *testing.T) {
	w := testWS(t)
	task, err := store.CreateTask(w, agentid.RootID, "core", "Needs an answer", store.TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	if err := store.MoveTask(w, task, model.StatusBlocked); err != nil {
		t.Fatalf("move: %v", err)
	}
	rec := liveRecord(t, w, "01RUNNOINDEXAGENT00000000", "streamy", task.Slug, "Looking at the file.\n[tool: Read]\n", 0)

	if got := Derive(w, rec, nil); got != Acting {
		t.Errorf("Derive = %q, want acting (no index to check blocked status)", got)
	}
}
