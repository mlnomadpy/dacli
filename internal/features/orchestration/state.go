package orchestration

import (
	"errors"
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
	_ = writeStateFile(path, body)
}

// writeStateFile replaces path's contents ATOMICALLY — temp file in the same
// directory, then rename, the way mdstore.WriteFile does. The loop's state
// files are written at every checkpoint from a process the operator kills with
// a signal, and they live in a repo full of concurrently running child agents:
// os.WriteFile truncates first, so an interrupted write (or a reader that
// arrives mid-write) leaves a file whose surviving fields read as ZERO —
// resetting exactly the token ceiling and thrash streak the file exists to
// carry across a restart (dacli 207). A rename either happened or it did not.
func writeStateFile(path, body string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".dacli-tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.WriteString(body); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		// Never orphan the temp file next to the real one — this path runs at
		// every checkpoint of every cycle.
		os.Remove(name)
		return err
	}
	return nil
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
	_ = writeStateFile(path, body)
}

// errCorruptState marks a governor snapshot that EXISTS but does not parse or
// does not make sense. It is deliberately distinct from the not-exist error a
// first run gets: "no snapshot yet" means start fresh, while "this snapshot is
// garbage" must never quietly mean the same thing (dacli 207) — the caller
// refuses the run instead, since resuming from zeroes is precisely the state
// that defeats the token ceiling and the thrash guard.
var errCorruptState = errors.New("corrupt governor state")

// readGovernorState loads the persisted governor snapshot for project,
// erroring if the loop has never checkpointed for it — the caller treats
// that as "start fresh", not a fault.
//
// Every field is VALIDATED rather than best-effort parsed. This file is plain
// `key: value` text sitting inside the repository the loop's own child agents
// are editing, and the previous parse discarded every error: a truncated write
// or a child writing `window_spent: 0` restored zeroes for the counters that
// were persisted specifically to survive a restart, silently resetting the
// guards. A snapshot that is missing a field, unparseable, negative, or
// dated in the future is refused instead (dacli 207).
func readGovernorState(w *workspace.Workspace, project string) (governorState, error) {
	path := governorStateFile(w, project)
	raw, err := os.ReadFile(path)
	if err != nil {
		return governorState{}, err
	}

	var st governorState
	seen := map[string]bool{}
	bad := func(format string, a ...any) (governorState, error) {
		return governorState{}, fmt.Errorf("%w (%s): %s", errCorruptState, path, fmt.Sprintf(format, a...))
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			// A line with no separator is a torn write, not a comment.
			return bad("malformed line %q", line)
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "cycle":
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return bad("cycle: %q is not a non-negative integer", v)
			}
			st.Cycle = n
		case "window_start":
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return bad("window_start: %q is not an RFC3339 timestamp", v)
			}
			// A future window start would make the rolling window never elapse
			// and WindowRemaining park the loop for the difference — a stopped
			// clock is not a budget.
			if t.After(time.Now().Add(time.Minute)) {
				return bad("window_start %s is in the future", v)
			}
			st.WindowStart = t
		case "window_spent":
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil || n < 0 {
				return bad("window_spent: %q is not a non-negative integer", v)
			}
			st.WindowSpent = n
		case "zero_streak":
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return bad("zero_streak: %q is not a non-negative integer", v)
			}
			st.ZeroStreak = n
		default:
			continue // forward-compatible: an unknown key is not corruption
		}
		seen[k] = true
	}
	for _, k := range []string{"cycle", "window_start", "window_spent", "zero_streak"} {
		if !seen[k] {
			return bad("missing %s", k)
		}
	}
	return st, nil
}
