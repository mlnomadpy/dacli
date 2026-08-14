package orchestration

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// The cycle journal persists the loop's LANDING ledger across invocations.
//
// Why it exists: the loop's default mode checkpoints and RETURNS after every
// cycle (only --yolo keeps one process alive), while `pendingAccept` and
// `pendingLand` lived in the driver struct. So in the documented mode, every
// cycle boundary destroyed them, and the next invocation:
//
//   - never ran `accept --force` for a PR that merged after the checkpoint, so
//     the task stayed open, was re-ranked, and a second implementer was spawned
//     onto already-landed work (observed repeatedly; issue #382 item 1);
//   - lost excludePending, so a task whose PR was still in flight was rebuilt;
//   - lost the held record push, so `main` advanced under queued PRs — the
//     stranded-PR bug of issue #75, resurrected on every restart.
//
// The three guarantees the loop advertises — accept only on confirmed merge,
// hold the record push while PRs are in flight, and never rebuild in-flight
// work — were therefore true only in --yolo and quietly false by default.
//
// It is a SEPARATE file from the governor snapshot on purpose: the governor
// holds counters whose corruption must refuse the run (resuming with reset
// guards defeats the token ceiling), while the journal holds a work ledger
// whose worst case is redoing a reconciliation that is already idempotent.
// A torn journal therefore degrades to "reconcile nothing this cycle" instead
// of halting the loop, and says so.
func journalFile(w *workspace.Workspace, project string) string {
	return filepath.Join(w.Root, workspace.Dir, "loop", project+"-journal.txt")
}

// cycleJournal is the cross-invocation half of the driver's landing state.
type cycleJournal struct {
	// PendingAccept are built tasks whose `accept --force` awaits confirmation
	// that their PR actually merged.
	PendingAccept []pendingAccept
	// PendingLand are self-PR record branches opened but not yet confirmed
	// merged; while any is in flight the record push is held.
	PendingLand []string
	// WindowTokens is the token CEILING (not the spend, which the governor
	// snapshot owns). Persisting it closes a silent-uncapping hole: the ceiling
	// came only from the --window-tokens flag, so a restart that omitted the
	// flag restored the spend and then ran with no cap at all.
	WindowTokens int64
	// Landing is the already-resolved policy for this bounded run. Keeping it
	// here prevents a restart between push, PR creation, checks, and merge from
	// selecting a different landing path because flags were omitted.
	Landing         model.LandingPolicy
	LandingExplicit bool
}

func (j cycleJournal) empty() bool {
	return len(j.PendingAccept) == 0 && len(j.PendingLand) == 0 && j.WindowTokens == 0 && j.Landing.Mode == ""
}

// writeCycleJournal persists the ledger, overwriting any prior one. An empty
// ledger REMOVES the file rather than writing a blank: absence then
// unambiguously means "nothing outstanding", so a stale file can never hold a
// record push hostage after the work it described has landed.
func writeCycleJournal(w *workspace.Workspace, project string, j cycleJournal) {
	path := journalFile(w, project)
	if j.empty() {
		_ = os.Remove(path)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "window_tokens: %d\n", j.WindowTokens)
	if j.Landing.Mode != "" {
		fmt.Fprintf(&b, "landing_mode: %s\n", j.Landing.Mode)
		fmt.Fprintf(&b, "landing_base: %s\n", j.Landing.Base)
		fmt.Fprintf(&b, "landing_override: %t\n", j.LandingExplicit)
	}
	for _, p := range j.PendingAccept {
		// seq and branch are joined by a space; a branch name cannot contain
		// one (git refuses it), so the split back is unambiguous.
		fmt.Fprintf(&b, "pending_accept: %d %s\n", p.Seq, p.Branch)
	}
	for _, br := range j.PendingLand {
		fmt.Fprintf(&b, "pending_land: %s\n", br)
	}
	_ = writeStateFile(path, b.String())
}

// readCycleJournal loads the ledger. A missing file is not an error — it is
// the common case (nothing outstanding). A malformed line is skipped with the
// reason returned in warn, so the loop can say what it dropped instead of
// silently reconciling a subset: unlike the governor snapshot, a partial
// ledger is safe (accept --force and the merge check are both idempotent),
// so degrading beats refusing the whole run.
func readCycleJournal(w *workspace.Workspace, project string) (j cycleJournal, warn []string) {
	raw, err := os.ReadFile(journalFile(w, project))
	if err != nil {
		return cycleJournal{}, nil
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			warn = append(warn, fmt.Sprintf("malformed line %q", line))
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "window_tokens":
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil || n < 0 {
				warn = append(warn, fmt.Sprintf("window_tokens %q is not a non-negative integer", v))
				continue
			}
			j.WindowTokens = n
		case "landing_mode":
			j.Landing.Mode = model.LandingMode(v)
		case "landing_base":
			j.Landing.Base = v
		case "landing_override":
			value, err := strconv.ParseBool(v)
			if err != nil {
				warn = append(warn, fmt.Sprintf("landing_override %q is not a boolean", v))
				continue
			}
			j.LandingExplicit = value
		case "pending_accept":
			seqStr, branch, ok := strings.Cut(v, " ")
			if !ok {
				warn = append(warn, fmt.Sprintf("pending_accept %q is not `<seq> <branch>`", v))
				continue
			}
			seq, err := strconv.Atoi(seqStr)
			if err != nil || seq <= 0 {
				warn = append(warn, fmt.Sprintf("pending_accept seq %q is not a positive integer", seqStr))
				continue
			}
			if branch = strings.TrimSpace(branch); branch == "" {
				warn = append(warn, fmt.Sprintf("pending_accept %q names no branch", v))
				continue
			}
			j.PendingAccept = append(j.PendingAccept, pendingAccept{Seq: seq, Branch: branch})
		case "pending_land":
			if v == "" {
				warn = append(warn, "pending_land names no branch")
				continue
			}
			j.PendingLand = append(j.PendingLand, v)
		default:
			warn = append(warn, fmt.Sprintf("unknown key %q", k))
		}
	}
	if j.Landing.Mode != "" {
		if err := model.ValidateLandingPolicy(j.Landing); err != nil {
			warn = append(warn, fmt.Sprintf("landing policy is invalid: %v", err))
			j.Landing = model.LandingPolicy{}
			j.LandingExplicit = false
		}
	}
	return j, warn
}
