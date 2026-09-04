package execution

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func runLifecycleLive(w *workspace.Workspace, rec procmon.Record, now time.Time) (bool, string) {
	// proc.txt is the terminal lifecycle authority shared by recovery and wait.
	// agents can stamp this outcome while outcome.md still says running and a
	// detached writer still advances its transcript; neither secondary signal
	// may resurrect the completed record (task 436).
	if rec.Outcome != "" {
		return false, ""
	}
	if raw, err := os.ReadFile(filepath.Join(w.RunDir(rec.RunID), "outcome.md")); err == nil {
		first, _, _ := strings.Cut(string(raw), "\n")
		if strings.HasPrefix(first, "outcome:") && first != detachedRunningPlaceholder {
			return false, ""
		}
	}
	// A durable watchdog verdict outranks every inferred liveness signal. In
	// particular, the timeout marker can be written while the run is still
	// young enough for startup grace; retaining it here leaks the task's path
	// claim even though the watchdog already killed and finalized the tree.
	if _, err := os.Stat(filepath.Join(w.RunDir(rec.RunID), timeoutMarker)); err == nil {
		return false, ""
	}
	// A governed kill records this marker only after the process tree has been
	// reaped. Reconcile the recorded identity once more before trusting it: the
	// marker is durable intent, while the process check prevents a partial or
	// forged marker from declaring a still-running worker terminal.
	if _, err := os.Stat(filepath.Join(w.RunDir(rec.RunID), "killed.txt")); err == nil && !runStillLive(rec) {
		return false, ""
	}
	// The guardian writes this only after Wait returns, so it is stronger
	// termination evidence than a process-table miss. Check it before transcript
	// activity: the final write commonly lands immediately before this marker.
	if _, err := os.Stat(filepath.Join(w.RunDir(rec.RunID), "runtime-exit.txt")); err == nil {
		return false, ""
	}
	if runStillLive(rec) {
		return true, "process live"
	}
	// For an unobservable identity the configured deadline is the finite upper
	// bound on transcript-derived liveness. The watchdog normally leaves a
	// marker, but recovery must remain correct if it could not do so.
	if rec.Timeout > 0 && !rec.Started.IsZero() && !now.Before(rec.Started.Add(rec.Timeout)) {
		return false, ""
	}
	if age := now.Sub(rec.Started); !rec.Started.IsZero() && age >= 0 && age < runStartupGrace {
		return true, "startup grace"
	}
	if info, err := os.Stat(filepath.Join(w.RunDir(rec.RunID), "transcript.log")); err == nil && info.Size() > 0 {
		if age := now.Sub(info.ModTime()); age >= 0 && age < transcriptActiveGrace {
			return true, "transcript active"
		}
		// Issue #672 showed transcript gaps longer than the fixed freshness
		// window while Codex was still editing and testing. Once output proves a
		// worker started, retain that evidence through the configured runtime
		// timeout unless the guardian records its real exit above. Legacy records
		// without a timeout deliberately retain the bounded freshness behavior.
		if rec.Timeout > 0 && !rec.Started.IsZero() && now.Before(rec.Started.Add(rec.Timeout)) {
			return true, "transcript active"
		}
	}
	return false, ""
}

func cmdWait(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("interval", "timeout"); err != nil {
		return err
	}
	result := commandresult.Wait{}
	ctx.Result = result
	interval := 3 * time.Second
	if n, err := f.Int("interval", 0); err != nil {
		return err
	} else if n > 0 {
		interval = time.Duration(n) * time.Second
	}
	overall := 3600
	if n, err := f.Int("timeout", 0); err != nil {
		return err
	} else if n > 0 {
		overall = n
	}

	pending := map[string]procmon.Record{}
	if len(f.Pos) > 0 {
		for _, ref := range f.Pos {
			if rec, ok := readProcByRef(w, ref); ok {
				// A recovered process miss still needs effects-based finalization.
				// Every other proc outcome is already terminal; waiting on it again
				// must not duplicate retirement, outcome writes, or exit events.
				if rec.Outcome == "" || rec.Outcome == recoveredExitOutcome {
					pending[rec.RunID] = rec
				}
			} else {
				fmt.Fprintf(ctx.Stderr, "no run matching %q\n", ref)
			}
		}
	} else {
		live, err := liveAgents(w)
		if err != nil {
			return err
		}
		for _, rec := range live {
			pending[rec.RunID] = rec
		}
	}
	if len(pending) == 0 {
		fmt.Fprintln(ctx.Stdout, "nothing to wait for")
		return nil
	}

	// Startup line: name how many runs we are waiting on and their short ids, so a
	// foreground wait shows what it is blocking on the moment it begins.
	total := len(pending)
	ids := make([]string, 0, total)
	for id := range pending {
		ids = append(ids, id[:min(10, len(id))])
	}
	sort.Strings(ids)
	fmt.Fprintf(ctx.Stdout, "waiting on %d run(s): %s\n", total, strings.Join(ids, ", "))

	// Light heartbeat: between completions the loop is silent for the whole
	// interval gap, so a long wait looks dead. Every ~30s (not every poll) print
	// one line proving the wait is still alive, without spamming.
	start := time.Now()
	nextBeat := start.Add(30 * time.Second)
	deadline := start.Add(time.Duration(overall) * time.Second)
	var runFailures []error
	for len(pending) > 0 {
		pendingIDs := make([]string, 0, len(pending))
		for id := range pending {
			pendingIDs = append(pendingIDs, id)
		}
		sort.Strings(pendingIDs)
		for _, id := range pendingIDs {
			rec := pending[id]
			// A run that raised the BLOCKED channel is finalized immediately, even
			// while its process is still live: it has told us it is stuck and will
			// not self-complete, so waiting on it as if it might is precisely the
			// silence task 269 removes. finalizeRun reports it as BLOCKED.
			if live, _ := runLifecycleLive(w, rec, lifecycleNow()); !live || readBlocked(w, id) != "" {
				summary, finalizeErr := finalizeRunChecked(w, rec)
				if finalizeErr != nil {
					return finalizeErr
				}
				fmt.Fprintf(ctx.Stdout, "%s  %s (%d of %d)\n", id[:min(10, len(id))], summary, total-len(pending)+1, total)
				outcome := "finalized"
				if completed, readErr := procmon.ReadRecord(filepath.Join(w.RunDir(id), "proc.txt")); readErr == nil && completed.Outcome != "" {
					outcome = completed.Outcome
				}
				result.Runs = append(result.Runs, commandresult.WaitRun{RunID: id, Child: rec.Child, Outcome: outcome})
				ctx.Result = result
				if failure := detachedRuntimeFailure(w, rec); failure != nil {
					runFailures = append(runFailures, fmt.Errorf("detached run %s: %w", id[:min(10, len(id))], failure))
				}
				delete(pending, id)
			}
		}
		if len(pending) == 0 {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("wait timed out with %d run(s) still live (raise --timeout, or dacli kill them)", len(pending))
		}
		if now := time.Now(); now.After(nextBeat) {
			fmt.Fprintf(ctx.Stdout, "still waiting on %d run(s) (up %s)\n", len(pending), now.Sub(start).Round(time.Second))
			nextBeat = now.Add(30 * time.Second)
		}
		time.Sleep(interval)
	}
	return errors.Join(runFailures...)
}

// detachedRuntimeFailure reconstructs the governed provider error after a
// detached guardian has exited. The guardian's numeric marker and bounded tail
// are durable; an exec.ExitError is not, so commandresult supplies a recorded
// exit cause with the same stable diagnostic fields (issue #876).
func detachedRuntimeFailure(w *workspace.Workspace, rec procmon.Record) error {
	rawExit, readErr := os.ReadFile(filepath.Join(w.RunDir(rec.RunID), "runtime-exit.txt"))
	if readErr != nil {
		if errors.Is(readErr, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read detached runtime exit marker: %w", readErr)
	}
	var exitCode int
	if _, scanErr := fmt.Sscanf(strings.TrimSpace(string(rawExit)), "%d", &exitCode); scanErr != nil {
		return fmt.Errorf("parse detached runtime exit marker: %w", scanErr)
	}
	if exitCode == 0 {
		return nil
	}
	binary := rec.Runtime
	if rt, err := store.LoadRuntime(w, rec.Runtime); err == nil && rt.Binary != "" {
		binary = rt.Binary
	}
	if binary == "" {
		binary = "runtime"
	}
	cmd := exec.Command(binary)
	cmd.Dir = w.Root
	tail, tailErr := readRuntimeDiagnosticTail(filepath.Join(w.RunDir(rec.RunID), "transcript.log"))
	externalErr := commandresult.NewRecordedExitError(cmd, commandresult.RunOptions{
		Operation: "runtime " + rec.Runtime + " detached launch", WorkspaceRoot: w.Root,
	}, nil, tail, exitCode)
	if tailErr != nil {
		return fmt.Errorf("read detached runtime transcript tail: %w", errors.Join(tailErr, externalErr))
	}
	return externalErr
}

func readRuntimeDiagnosticTail(path string) ([]byte, error) {
	const limit = 8 << 10
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > limit {
		if _, seekErr := f.Seek(info.Size()-limit, io.SeekStart); seekErr != nil {
			return nil, seekErr
		}
	}
	return io.ReadAll(io.LimitReader(f, limit))
}

// readProcByRef finds any run (live or finished) whose id-prefix or child id
// matches ref.
func readProcByRef(w *workspace.Workspace, ref string) (procmon.Record, bool) {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return procmon.Record{}, false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(e.Name()), "proc.txt"))
		if err != nil {
			continue
		}
		if strings.HasPrefix(rec.RunID, ref) || rec.Child == ref {
			return rec, true
		}
	}
	return procmon.Record{}, false
}

// finalizeRun computes a finished detached run's outcome from what it actually
// wrote to the workspace (acceptance boxes + events by the child), overwriting
// the "running (detached)" placeholder. A detached child is not our OS child,
// so there is no exit code to read — the outcome is derived from effects, which
// is the honest thing to report.
func finalizeRun(w *workspace.Workspace, rec procmon.Record) string {
	summary, err := finalizeRunChecked(w, rec)
	if err != nil {
		return fmt.Sprintf("%s: finalization failed: %v", rec.Child, err)
	}
	return summary
}

func finalizeRunChecked(w *workspace.Workspace, rec procmon.Record) (string, error) {
	runDir := w.RunDir(rec.RunID)
	record := openRunRecord(runDir, nil)
	workDir := w.Root
	isolatedWorktree := false
	if raw, e := os.ReadFile(filepath.Join(runDir, "worktree.txt")); e == nil {
		candidate := strings.TrimSpace(string(raw))
		if candidate != "" {
			workDir, isolatedWorktree = candidate, true
		}
	}
	var plannedHandoffs []string
	if raw, err := os.ReadFile(filepath.Join(runDir, "planned-handoffs.txt")); err == nil && strings.TrimSpace(string(raw)) != "" {
		for _, line := range strings.Split(string(raw), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				plannedHandoffs = append(plannedHandoffs, line)
			}
		}
	}
	plannedHandoff := len(plannedHandoffs) > 0
	// The independent watchdog owns the timed-out verdict. A concurrently
	// polling `wait` may observe the now-dead tree immediately afterwards; it
	// must not overwrite that durable verdict with effects-derived "done" or
	// "no visible result" (task 372).
	if _, err := os.Stat(filepath.Join(runDir, timeoutMarker)); err == nil {
		if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, "timed out"); err != nil {
			return "", fmt.Errorf("record critical run artifact proc.txt: %w", err)
		}
		if rec.Child != "" {
			_ = store.RetireAgent(w, rec.Child)
		}
		return fmt.Sprintf("%s: timed out after %s", rec.Child, rec.Timeout), nil
	}
	// Free the agent's WIP slot only after its terminal artifacts are durable.
	// Nothing in the
	// lifecycle called RetireAgent, so every spawn's agent held capacity
	// forever: roles filled to their limit and later spawns were refused while
	// `dacli agents` showed nobody live (task 282). ActiveInRole no longer
	// COUNTS a finished agent either — that half is self-healing for the
	// backlog already leaked — but retiring here keeps the roster honest about
	// which agents are done rather than merely inferred-done.
	//
	// The retirement itself remains best-effort: an agent file that cannot be
	// written must never invalidate an otherwise durable terminal run record.
	// The break-glass BLOCKED channel wins over any derived outcome: a child that
	// raised it told us, in its own words, that it could not run dacli. Reporting
	// that run as "done" or "no visible result" would bury exactly the failure the
	// channel exists to surface, so BLOCKED is stamped and returned first (269).
	if reason := readBlocked(w, rec.RunID); reason != "" {
		elapsed := time.Since(rec.Started).Round(time.Second)
		if isolatedWorktree || store.RootHandoffRequested(w, rec.RunID) {
			handoff, required, captureErr := store.CaptureRootHandoff(w, rec.RunID, rec.Task, rec.Child, workDir, store.RootHandoffRequest{
				Schema: store.RootHandoffSchema, FailedOperation: "worker lifecycle publication", FailureClass: "policy_refusal", Stderr: reason,
				NextAction: "owner consumes the handoff after hash re-observation, reruns verification, then commits and publishes without changing worker harness or grant",
			}, time.Now())
			if captureErr != nil {
				return "", fmt.Errorf("capture root handoff: %w", captureErr)
			}
			if required {
				if err := record.critical("outcome.md", fmt.Sprintf("outcome: handoff-required (detached)\nchild: %s\nelapsed_since_start: %s\nfailed_operation: %s\nnext: %s\n", rec.Child, elapsed, handoff.FailedOperation, handoff.NextAction)); err != nil {
					return "", err
				}
				if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, "handoff-required"); err != nil {
					return "", fmt.Errorf("record critical run artifact proc.txt: %w", err)
				}
				if rec.Child != "" {
					_ = store.RetireAgent(w, rec.Child)
				}
				recordExit(w, rec, "handoff-required", elapsed, handoff.FailedOperation)
				return fmt.Sprintf("%s: handoff-required — %s", rec.Child, handoff.NextAction), nil
			}
		}
		if err := record.critical("outcome.md", fmt.Sprintf("outcome: blocked (detached)\nchild: %s\nelapsed_since_start: %s\nreason: %s\n",
			rec.Child, elapsed, reason)); err != nil {
			return "", err
		}
		if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, "blocked"); err != nil {
			return "", fmt.Errorf("record critical run artifact proc.txt: %w", err)
		}
		if rec.Child != "" {
			_ = store.RetireAgent(w, rec.Child)
		}
		recordExit(w, rec, "blocked", elapsed, fmt.Sprintf("the child raised the BLOCKED channel: %s", firstLine(reason)))
		return fmt.Sprintf("%s: BLOCKED — %s", rec.Child, firstLine(reason)), nil
	}
	eventsWS := w
	if isolatedWorktree {
		raw := []byte(workDir)
		if wtw, e2 := workspace.Find(strings.TrimSpace(string(raw))); e2 == nil {
			eventsWS = wtw
		}
	}
	done, total := 0, 0
	if t, _ := store.FindTask(w, rec.Task); t != nil {
		for _, b := range t.Acceptance() {
			total++
			if b.Done {
				done++
			}
		}
	}
	childEvents, _ := eventlog.List(eventsWS, eventlog.Query{Actor: rec.Child})
	providerSummary := ""
	if _, err := os.Stat(filepath.Join(runDir, "provider-outcome.txt")); os.IsNotExist(err) {
		if raw, readErr := os.ReadFile(filepath.Join(runDir, "runtime-exit.txt")); readErr == nil {
			var exitCode int
			if _, scanErr := fmt.Sscanf(strings.TrimSpace(string(raw)), "%d", &exitCode); scanErr == nil && exitCode != 0 {
				var printed strings.Builder
				if policyErr := recordProviderOutcome(&printed, w, rec.Runtime, filepath.Join(runDir, "transcript.log"), exitCode, record.bestEffort); policyErr != nil {
					providerSummary = fmt.Sprintf("provider policy record failed: %v", policyErr)
				} else {
					providerSummary = strings.TrimSpace(printed.String())
				}
			}
		}
	}
	// A detached child streamed straight to transcript.log without an in-process
	// parser (the parent had already returned), so usage was never captured live.
	// If the transcript is a stream-json log, harvest its final usage now. Parsing
	// is self-detecting: a plain-text transcript yields no `result` event and
	// nothing is written, so text runtimes are unaffected.
	if _, err := os.Stat(filepath.Join(runDir, "usage.txt")); os.IsNotExist(err) {
		if f, e := os.Open(filepath.Join(runDir, "transcript.log")); e == nil {
			u := teeStreamJSON(f, io.Discard)
			if !u.found {
				_, _ = f.Seek(0, io.SeekStart)
				u = teeStructuredJSON(f, io.Discard, "codex-jsonl")
			}
			_ = f.Close()
			if u.found {
				writeUsage(runDir, u)
			}
		}
	}
	var claimProjectionErr error
	claimSandboxed := false
	if _, err := os.Stat(claimSandboxPath(w, rec.RunID)); err == nil {
		claimSandboxed = true
		rawExit, readErr := os.ReadFile(filepath.Join(runDir, "runtime-exit.txt"))
		var exitCode int
		if readErr != nil {
			claimProjectionErr = fmt.Errorf("read claim sandbox runtime exit: %w", readErr)
		} else if _, scanErr := fmt.Sscanf(strings.TrimSpace(string(rawExit)), "%d", &exitCode); scanErr != nil {
			claimProjectionErr = fmt.Errorf("parse claim sandbox runtime exit: %w", scanErr)
		} else if exitCode == 0 {
			_, claimProjectionErr = projectClaimSandbox(w, rec.RunID, time.Now())
		}
	}
	elapsed := time.Since(rec.Started).Round(time.Second)
	outcome := "done"
	var handoff store.RootHandoff
	handoffRequired := false
	if store.RootHandoffRequested(w, rec.RunID) || (isolatedWorktree && (plannedHandoff || claimSandboxed)) {
		var captureErr error
		handoff, handoffRequired, captureErr = store.CaptureRootHandoff(w, rec.RunID, rec.Task, rec.Child, workDir, store.RootHandoffRequest{
			Schema: store.RootHandoffSchema, FailedOperation: "worker lifecycle publication", FailureClass: "filesystem_sandbox_refusal",
			NextAction: "owner consumes the handoff after hash re-observation, reruns verification, then commits and publishes without changing worker harness or grant",
		}, time.Now())
		if captureErr != nil {
			return "", fmt.Errorf("capture root handoff: %w", captureErr)
		}
	}
	if handoffRequired {
		commitHandoffs := append([]string(nil), plannedHandoffs...)
		if claimSandboxed {
			commitHandoffs = append(commitHandoffs, "git-metadata-write:claim-sandbox")
		}
		if _, resolved, _ := applyParentCommitIfPlanned(w, handoff, commitHandoffs, time.Now()); resolved {
			handoffRequired = false
		}
	}
	if claimProjectionErr != nil {
		outcome = "failed"
	} else if handoffRequired {
		outcome = "handoff-required"
	} else if len(childEvents) == 0 && done == 0 {
		outcome = "no visible result"
	}
	if err := record.critical("outcome.md", fmt.Sprintf("outcome: %s (detached)\nchild: %s\nelapsed_since_start: %s\nacceptance: %d/%d\nevents_by_child: %d\n",
		outcome, rec.Child, elapsed, done, total, len(childEvents))); err != nil {
		return "", err
	}
	if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, outcome); err != nil {
		return "", fmt.Errorf("record critical run artifact proc.txt: %w", err)
	}
	if rec.Child != "" {
		_ = store.RetireAgent(w, rec.Child)
	}
	recordExit(w, rec, outcome, elapsed, fmt.Sprintf("wrote %d event(s), checked %d of %d acceptance box(es)",
		len(childEvents), done, total))
	summary := fmt.Sprintf("%s: %s · %s · %d event(s) · acceptance %d/%d",
		rec.Child, outcome, elapsed, len(childEvents), done, total)
	if providerSummary != "" {
		return providerSummary + " · " + summary, nil
	}
	if handoffRequired {
		summary += " · next: " + handoff.NextAction
	}
	if claimProjectionErr != nil {
		return summary, fmt.Errorf("claim sandbox projection: %w", claimProjectionErr)
	}
	return summary, nil
}

// recordExit writes the run's ending into the append-only log, so the fact that
// an agent finished — and with what result — survives independently of who
// later reads a run directory (issue #449).
//
// finalizeRun is reached exactly once per run: it is gated on an outcome that
// is missing or still the running placeholder, and it overwrites that outcome
// before returning. So the event is written once, not once per observation.
//
// Best-effort. A run whose ending cannot be logged is still finalized —
// refusing to finalize because the log write failed would restore precisely the
// invisible-run state this exists to end.
func recordExit(w *workspace.Workspace, rec procmon.Record, outcome string, elapsed time.Duration, detail string) {
	if rec.Child == "" {
		return // nothing to attribute the ending to
	}
	body := fmt.Sprintf("run %s ended: %s after %s — %s", rec.RunID, outcome, elapsed, detail)
	if rec.Role != "" {
		body += fmt.Sprintf(" (role %s)", rec.Role)
	}
	_, _ = eventlog.Append(w, rec.Child, model.EventExit, rec.Task, "run", body)
}

// detachedRunningPlaceholder is the exact first line of the outcome.md a
// `spawn --detach` writes before the run has finished. finalizeRun overwrites
// it; any run still holding it whose process is gone was never finalized —
// nobody ran `dacli wait` on it.
const detachedRunningPlaceholder = "outcome: running (detached)"

// humanKB renders a KB resident-set size as MiB/GiB.
func humanKB(kb int) string {
	mb := float64(kb) / 1024
	if mb >= 1024 {
		return fmt.Sprintf("%.1fGiB", mb/1024)
	}
	return fmt.Sprintf("%.0fMiB", mb)
}

// gpuStr renders GPU memory, honestly reporting n/a where it cannot be
// measured (no nvidia-smi) rather than a misleading 0.
func gpuStr(mib int) string {
	if mib < 0 {
		return "n/a"
	}
	return fmt.Sprintf("%dMiB", mib)
}

// warnMainCheckoutSpawn reports the two ways a no-worktree spawn leaves the
// repository unable to land work: no trunk to integrate into, or a main
// checkout already parked on a task branch.
//
// It warns rather than refuses. A no-worktree spawn is legitimate — a
// single-agent run, a read-only reviewer, a repo with no git at all — so
// refusing would break working setups to prevent a mistake the operator may
// not be making. What is not acceptable is silence, which is what produced a
// trunkless repo and an "integrated 0" that read as success.
