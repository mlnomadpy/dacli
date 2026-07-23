package orchestration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
)

// TestLoopPersistsStateForStatusToRead is the 093 regression: a completed
// cycle must leave behind a snapshot that `dacli loop status` can read back —
// cycle count, trunk marker, tokens spent this window, and the ready backlog
// size — without needing the loop process itself still running.
func TestLoopPersistsStateForStatusToRead(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}

	fr := &fakeRunner{}
	gov := &Governor{MaxCycles: 1, NoProgressHalt: 3}
	d := newDriver(w, fr, gov)
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	st, err := readLoopState(w, "p")
	if err != nil {
		t.Fatalf("expected a persisted loop state, got error: %v", err)
	}
	if st.Cycle != 1 {
		t.Fatalf("want cycle 1 persisted, got %d", st.Cycle)
	}
	if st.Status == "" {
		t.Fatal("want a persisted status decision, got empty")
	}
	if st.UpdatedAt.IsZero() {
		t.Fatal("want a persisted timestamp")
	}

	// `dacli loop status` reads the same snapshot back through the command
	// surface, not just the internal helper.
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdLoopStatus(ctx, nil); err != nil {
		t.Fatalf("loop status: %v", err)
	}
	got := out.String()
	for _, want := range []string{"cycle 1", "trunk marker", "tokens this window", "ready backlog"} {
		if !strings.Contains(got, want) {
			t.Fatalf("loop status output missing %q, got: %s", want, got)
		}
	}
}

// TestLoopStatusErrorsWithoutAPriorLoopRun covers the honest-degrade path: no
// persisted state means the command must say so, not print zeroes as if a
// loop had actually run.
func TestLoopStatusErrorsWithoutAPriorLoopRun(t *testing.T) {
	w := loopEnv(t)
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdLoopStatus(ctx, nil); err == nil {
		t.Fatal("want an error when no loop has ever run for the project")
	}
}
