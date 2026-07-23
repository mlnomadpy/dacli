package orchestration

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

// loopState is the latest snapshot of a loop run for one project, persisted to
// disk so `dacli loop status` can report on a loop that is still running in
// another process, or on the last completed run once the loop has exited. It
// is written at every governor checkpoint — best-effort, and never consulted
// by the loop's own control flow (the in-memory Governor remains the single
// source of truth while a loop is actually running).
type loopState struct {
	Project      string
	Cycle        int
	TrunkMarker  int
	WindowTokens int64
	Backlog      int
	Status       string // last governor decision: proceed, idle, sleep-window, halt
	Reason       string
	UpdatedAt    time.Time
}

func loopStateFile(w *workspace.Workspace, project string) string {
	return filepath.Join(w.Root, workspace.Dir, "loop", project+".txt")
}

// writeLoopState persists st, overwriting any prior snapshot for the project.
// Failures are swallowed: a status snapshot is a convenience, never load-
// bearing for the loop itself.
func writeLoopState(w *workspace.Workspace, st loopState) {
	path := loopStateFile(w, st.Project)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	body := fmt.Sprintf(
		"project: %s\ncycle: %d\ntrunk_marker: %d\nwindow_tokens: %d\nbacklog: %d\nstatus: %s\nreason: %s\nupdated_at: %s\n",
		st.Project, st.Cycle, st.TrunkMarker, st.WindowTokens, st.Backlog, st.Status, st.Reason,
		st.UpdatedAt.UTC().Format(time.RFC3339))
	_ = os.WriteFile(path, []byte(body), 0o644)
}

// readLoopState loads the persisted snapshot for project, erroring if the
// loop has never run (or never reached a checkpoint) for it.
func readLoopState(w *workspace.Workspace, project string) (loopState, error) {
	raw, err := os.ReadFile(loopStateFile(w, project))
	if err != nil {
		return loopState{}, err
	}
	st := loopState{Project: project}
	for _, line := range strings.Split(string(raw), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "cycle":
			st.Cycle, _ = strconv.Atoi(v)
		case "trunk_marker":
			st.TrunkMarker, _ = strconv.Atoi(v)
		case "window_tokens":
			n, _ := strconv.ParseInt(v, 10, 64)
			st.WindowTokens = n
		case "backlog":
			st.Backlog, _ = strconv.Atoi(v)
		case "status":
			st.Status = v
		case "reason":
			st.Reason = v
		case "updated_at":
			t, _ := time.Parse(time.RFC3339, v)
			st.UpdatedAt = t
		}
	}
	return st, nil
}

// governorStateFile is deliberately distinct from loopStateFile: the loop
// status snapshot is a convenience `dacli loop status` reads and the loop
// itself never consults; this file is the opposite — the loop's own control
// flow reloads it at startup so a restart resumes the governor's cycle
// count, budget window, and thrash streak instead of resetting them.
func governorStateFile(w *workspace.Workspace, project string) string {
	return filepath.Join(w.Root, workspace.Dir, "loop", project+"-governor.txt")
}

// writeGovernorState persists the governor's running counters, overwriting
// any prior snapshot for the project. Failures are swallowed the same way
// writeLoopState's are: a restart that finds nothing to reload simply starts
// fresh, which is the pre-existing behavior.
func writeGovernorState(w *workspace.Workspace, project string, st governorState) {
	path := governorStateFile(w, project)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	body := fmt.Sprintf(
		"cycle: %d\nwindow_start: %s\nwindow_spent: %d\nzero_streak: %d\n",
		st.Cycle, st.WindowStart.UTC().Format(time.RFC3339), st.WindowSpent, st.ZeroStreak)
	_ = os.WriteFile(path, []byte(body), 0o644)
}

// readGovernorState loads the persisted governor snapshot for project,
// erroring if the loop has never checkpointed for it — the caller treats
// that as "start fresh", not a fault.
func readGovernorState(w *workspace.Workspace, project string) (governorState, error) {
	raw, err := os.ReadFile(governorStateFile(w, project))
	if err != nil {
		return governorState{}, err
	}
	var st governorState
	for _, line := range strings.Split(string(raw), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "cycle":
			st.Cycle, _ = strconv.Atoi(v)
		case "window_start":
			t, _ := time.Parse(time.RFC3339, v)
			st.WindowStart = t
		case "window_spent":
			n, _ := strconv.ParseInt(v, 10, 64)
			st.WindowSpent = n
		case "zero_streak":
			st.ZeroStreak, _ = strconv.Atoi(v)
		}
	}
	return st, nil
}
