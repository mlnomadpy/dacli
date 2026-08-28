package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const loopRecoverySchema = "loop-recovery/v1"

type recoveryRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type recoveryObservation struct {
	Source     string                   `json:"source"`
	Ref        recoveryRef              `json:"ref"`
	State      string                   `json:"state"`
	Confidence store.DeliveryConfidence `json:"confidence"`
	Detail     string                   `json:"detail"`
}

// loopRecoveryCheckpoint is the durable machine-readable answer to “why did
// this bounded invocation stop, what exact evidence did it observe, and what
// is the smallest safe next action?”. It complements the governor snapshot;
// it never owns or resets cycle, budget, WIP, claim, STOP, or landing state.
type loopRecoveryCheckpoint struct {
	Schema       string                `json:"schema"`
	Version      int                   `json:"version"`
	Project      string                `json:"project"`
	Cycle        int                   `json:"cycle"`
	Checkpoint   string                `json:"checkpoint"`
	HaltClass    string                `json:"halt_class"`
	AffectedRefs []recoveryRef         `json:"affected_refs"`
	Observed     []recoveryObservation `json:"observed"`
	TrunkMarker  int                   `json:"trunk_marker"`
	TrunkKnown   bool                  `json:"trunk_known"`
	Retryable    bool                  `json:"retryable"`
	NextAction   string                `json:"next_action"`
	Reason       string                `json:"reason"`
	ObservedAt   time.Time             `json:"observed_at"`
}

func loopRecoveryFile(w *workspace.Workspace, project string) string {
	return filepath.Join(w.Root, workspace.Dir, "loop", project+"-recovery.json")
}

func writeLoopRecovery(w *workspace.Workspace, cp loopRecoveryCheckpoint) error {
	cp.Schema, cp.Version = loopRecoverySchema, 1
	raw, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}
	path := loopRecoveryFile(w, cp.Project)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeStateFile(path, string(append(raw, '\n')))
}

func readLoopRecovery(w *workspace.Workspace, project string) (loopRecoveryCheckpoint, error) {
	var cp loopRecoveryCheckpoint
	raw, err := os.ReadFile(loopRecoveryFile(w, project))
	if err != nil {
		return cp, err
	}
	if err := json.Unmarshal(raw, &cp); err != nil {
		return cp, fmt.Errorf("decode loop recovery checkpoint: %w", err)
	}
	if cp.Schema != loopRecoverySchema || cp.Version != 1 || cp.Project != project || cp.Checkpoint == "" || cp.HaltClass == "" {
		return cp, fmt.Errorf("invalid loop recovery checkpoint for project %s", project)
	}
	return cp, nil
}

func refKind(f store.DeliveryFinding) string {
	switch {
	case f.Classification == "github-state-unknown":
		return "project"
	case strings.HasPrefix(f.Classification, "handoff-") || strings.HasPrefix(f.Source, "run/"):
		return "run"
	case strings.Contains(f.Classification, "event"):
		return "event"
	case strings.Contains(f.Classification, "run"):
		return "run"
	case strings.Contains(f.Classification, "pr") || f.Source == "github":
		return "task"
	default:
		return "project"
	}
}

func findingBlocksNewWave(f store.DeliveryFinding) bool {
	switch f.Classification {
	case "finished-unfinalized-run", "terminal-task-event", "event-state-unknown", "task-agent-state-unknown", "handoff-required", "handoff-state-unknown":
		return true
	default:
		return false
	}
}

func recoveryFromFindings(project string, cycle int, trunk int, trunkKnown bool, observedAt time.Time, findings []store.DeliveryFinding) *loopRecoveryCheckpoint {
	if len(findings) == 0 {
		return nil
	}
	cp := &loopRecoveryCheckpoint{Project: project, Cycle: cycle, Checkpoint: "pre-cycle-reconciliation", HaltClass: "inconsistent-record", TrunkMarker: trunk, TrunkKnown: trunkKnown, ObservedAt: observedAt.UTC()}
	next := map[string]bool{}
	for _, f := range findings {
		kind := refKind(f)
		cp.AffectedRefs = append(cp.AffectedRefs, recoveryRef{Kind: kind, ID: f.ObjectID})
		for _, related := range f.RelatedRefs {
			cp.AffectedRefs = append(cp.AffectedRefs, recoveryRef{Kind: related.Kind, ID: related.ID})
		}
		cp.Observed = append(cp.Observed, recoveryObservation{Source: f.Source, Ref: recoveryRef{Kind: kind, ID: f.ObjectID}, State: firstNonempty(f.DiagnosisCode, f.Classification), Confidence: f.Confidence, Detail: f.Detail})
		if f.NextAction != "" {
			next[f.NextAction] = true
		}
		if f.Classification == "github-state-unknown" {
			cp.HaltClass = "transient-infrastructure-failure"
			cp.Retryable = f.Retryable
		} else if f.Classification == "handoff-required" {
			cp.HaltClass = "handoff-required"
			cp.Retryable = false
		} else if f.Classification == "verification-required" {
			cp.HaltClass = "policy-refusal"
			cp.Retryable = false
		} else if f.Source == "github" && cp.HaltClass != "transient-infrastructure-failure" {
			cp.HaltClass = "external-blocker"
			cp.Retryable = cp.Retryable || f.Retryable
		}
	}
	actions := make([]string, 0, len(next))
	for action := range next {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	cp.NextAction = strings.Join(actions, "; ")
	cp.Reason = fmt.Sprintf("previous-cycle reconciliation found %d blocker(s)", len(findings))
	sort.Slice(cp.AffectedRefs, func(i, j int) bool {
		if cp.AffectedRefs[i].Kind == cp.AffectedRefs[j].Kind {
			return cp.AffectedRefs[i].ID < cp.AffectedRefs[j].ID
		}
		return cp.AffectedRefs[i].Kind < cp.AffectedRefs[j].Kind
	})
	unique := cp.AffectedRefs[:0]
	for _, ref := range cp.AffectedRefs {
		if len(unique) == 0 || unique[len(unique)-1] != ref {
			unique = append(unique, ref)
		}
	}
	cp.AffectedRefs = unique
	return cp
}

func firstNonempty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "unknown"
}

// reconcileBeforeCycle observes local evidence on every invocation and GitHub
// whenever a durable landing entry makes GitHub authoritative. Unknown remote
// evidence is a halt, never an implicit green result. A later invocation
// observes the changed condition afresh; no counter is edited to enable it.
func (d *driver) reconcileBeforeCycle() (*loopRecoveryCheckpoint, error) {
	now := d.now()
	prior, priorErr := readLoopRecovery(d.w, d.cfg.project)
	projection, err := store.LocalDeliveryProjection(d.w, d.cfg.project, now)
	if err != nil {
		return nil, err
	}
	var blockers []store.DeliveryFinding
	observedExternal := false
	for _, finding := range projection.Findings {
		enrichHandoffFinding(d.w, &finding)
		if findingBlocksNewWave(finding) {
			blockers = append(blockers, finding)
		}
	}
	if len(d.pendingAccept) > 0 || len(d.pendingLand) > 0 {
		projection, observeErr := store.ReconcileDelivery(d.w, d.cfg.project, now)
		byTask := map[string]store.DeliveryFinding{}
		for _, finding := range projection.Findings {
			if finding.Source == "github" {
				byTask[finding.ObjectID] = finding
			}
			if finding.Classification == "github-state-unknown" {
				for _, pending := range d.pendingAccept {
					taskID := fmt.Sprintf("%03d", pending.Seq)
					if task, findErr := store.FindTask(d.w, taskID); findErr == nil {
						taskID = task.ID
					}
					finding.RelatedRefs = append(finding.RelatedRefs, store.DeliveryRef{Kind: "task", ID: taskID}, store.DeliveryRef{Kind: "branch", ID: pending.Branch})
				}
				for _, branch := range d.pendingLand {
					finding.RelatedRefs = append(finding.RelatedRefs, store.DeliveryRef{Kind: "branch", ID: branch})
				}
				blockers = append(blockers, finding)
			}
		}
		if observeErr == nil {
			observedExternal = true
			for _, pending := range d.pendingAccept {
				task, findErr := store.FindTask(d.w, fmt.Sprintf("%03d", pending.Seq))
				if findErr != nil {
					blockers = append(blockers, store.DeliveryFinding{Classification: "task-state-unknown", ObjectID: fmt.Sprintf("%03d", pending.Seq), Source: "task", Severity: "major", Confidence: store.DeliveryUnknown, Detail: findErr.Error(), NextAction: "restore the pending task record before resuming"})
					continue
				}
				finding, found := byTask[task.ID]
				if !found {
					blockers = append(blockers, store.DeliveryFinding{Classification: "missing-canonical-pr", ObjectID: task.ID, Source: "github", Severity: "major", Confidence: store.DeliveryKnown, DiagnosisCode: "missing_canonical_pr", Detail: "no pull request matched durable branch " + pending.Branch, NextAction: "push the canonical task branch and open a pull request for its exact head"})
					continue
				}
				if finding.DiagnosisCode == "ci_pending" {
					if !d.checkpointTaskPhase(task, phaseCIPending) {
						return nil, d.phaseErr
					}
				}
				if finding.Classification == "merged-pr-task-nonterminal" {
					if !d.checkpointTaskPhase(task, phaseMerged) {
						return nil, d.phaseErr
					}
				}
				// A merge is actionable by reconcilePendingAccepts below unless
				// command acceptance still requires owner verification. That is an
				// exit-3 policy boundary, not permission to start another wave.
				if finding.Classification == "merged-pr-task-nonterminal" && taskRequiresVerifierEvidence(task) && !(task.Status == model.StatusDone && acceptanceComplete(task)) {
					finding.Classification = "verification-required"
					finding.DiagnosisCode = "verification_required"
					finding.Source = "task"
					finding.NextAction = fmt.Sprintf("run dacli accept %03d --verify <command> as the workspace owner", task.Seq)
					finding.Retryable = false
					blockers = append(blockers, finding)
				} else if finding.Classification != "merged-pr-task-nonterminal" {
					blockers = append(blockers, finding)
				}
			}
		}
	}
	checkpoint := recoveryFromFindings(d.cfg.project, d.gov.Cycle(), d.lastTrunkMarker, d.lastTrunkKnown, now, blockers)
	resolvedHandoff := priorErr == nil && prior.Checkpoint == "pre-cycle-reconciliation" && prior.HaltClass == "handoff-required"
	if checkpoint == nil && (observedExternal || resolvedHandoff) && priorErr == nil && prior.Checkpoint == "pre-cycle-reconciliation" {
		resolved := fmt.Sprintf("observed prior %s resolved; resuming from cycle %d without resetting governor counters", prior.HaltClass, d.gov.Cycle())
		if d.recovery == "" {
			d.recovery = resolved
		} else if !strings.Contains(d.recovery, resolved) {
			d.recovery += "; " + resolved
		}
	}
	return checkpoint, nil
}

func enrichHandoffFinding(w *workspace.Workspace, finding *store.DeliveryFinding) {
	if finding.Classification != "handoff-required" && finding.Classification != "handoff-state-unknown" {
		return
	}
	handoff, err := store.LoadRootHandoff(w, finding.ObjectID)
	if err != nil {
		return
	}
	finding.RelatedRefs = append(finding.RelatedRefs,
		store.DeliveryRef{Kind: "task", ID: handoff.TaskID},
		store.DeliveryRef{Kind: "agent", ID: handoff.ChildID},
		store.DeliveryRef{Kind: "worktree", ID: handoff.Worktree},
	)
}

func decisionRecovery(d *driver, status, reason string) loopRecoveryCheckpoint {
	cp := loopRecoveryCheckpoint{Project: d.cfg.project, Cycle: d.gov.Cycle(), Checkpoint: "governor", TrunkMarker: d.lastTrunkMarker, TrunkKnown: d.lastTrunkKnown, Reason: reason, ObservedAt: d.now().UTC()}
	switch status {
	case Halt.String():
		cp.HaltClass = "policy-refusal"
		cp.NextAction = "satisfy the named policy condition, then invoke another bounded cycle"
	case SleepWindow.String():
		cp.HaltClass, cp.Retryable = "budget-exhaustion", true
		cp.NextAction = "wait for the persisted token window to reset"
	case Idle.String():
		cp.HaltClass, cp.Retryable = "no-schedulable-work", true
		cp.NextAction = "resolve blocked or claim-conflicting work, or add evidence-backed work"
	default:
		cp.HaltClass = "checkpoint"
		cp.NextAction = "continue from this checkpoint"
	}
	return cp
}
