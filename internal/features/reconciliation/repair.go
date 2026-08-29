package reconciliation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const repairPlanSchema = "reconciliation-repair/v1"

type repairPrecondition struct {
	FindingID      string                   `json:"finding_id"`
	Classification string                   `json:"classification"`
	ObjectID       string                   `json:"object_id"`
	Source         string                   `json:"source"`
	Confidence     store.DeliveryConfidence `json:"confidence"`
	Detail         string                   `json:"detail"`
	RelatedRefs    []store.DeliveryRef      `json:"related_refs,omitempty"`
}

type repairOperation struct {
	ID            string   `json:"id"`
	FindingIDs    []string `json:"finding_ids"`
	Mode          string   `json:"mode"` // delegated or manual
	DelegatedTo   string   `json:"delegated_to,omitempty"`
	DelegatedPlan string   `json:"delegated_plan,omitempty"`
	Argv          []string `json:"argv,omitempty"`
	Rollback      []string `json:"rollback"`
	NextAction    string   `json:"next_action"`
}

type repairPlan struct {
	Schema        string               `json:"schema"`
	Version       int                  `json:"version"`
	ID            string               `json:"id"`
	Project       string               `json:"project"`
	ObservedAt    time.Time            `json:"observed_at"`
	Preconditions []repairPrecondition `json:"preconditions"`
	Operations    []repairOperation    `json:"operations"`
}

type repairOperationAudit struct {
	ID          string    `json:"id"`
	State       string    `json:"state"`
	Detail      string    `json:"detail,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

type repairAudit struct {
	Schema     string                 `json:"schema"`
	PlanID     string                 `json:"plan_id"`
	Project    string                 `json:"project"`
	StartedAt  time.Time              `json:"started_at"`
	Operations []repairOperationAudit `json:"operations"`
}

func planRepairs(w *workspace.Workspace, project string, now time.Time) (repairPlan, error) {
	delivery, observeErr := store.ReconcileDelivery(w, project, now)
	plan := repairPlan{Schema: repairPlanSchema, Version: 1, Project: project, ObservedAt: now.UTC(), Preconditions: []repairPrecondition{}, Operations: []repairOperation{}}
	for _, finding := range delivery.Findings {
		plan.Preconditions = append(plan.Preconditions, repairPrecondition{FindingID: finding.ID, Classification: finding.Classification, ObjectID: finding.ObjectID, Source: finding.Source, Confidence: finding.Confidence, Detail: finding.Detail, RelatedRefs: append([]store.DeliveryRef(nil), finding.RelatedRefs...)})
	}
	cleanupPlan, cleanupErr := store.PlanRepositoryCleanup(w, project, now, w.Root)
	if cleanupErr == nil {
		eligible := false
		for _, item := range cleanupPlan.Items {
			eligible = eligible || item.Eligible
		}
		for _, artifact := range cleanupPlan.Artifacts {
			eligible = eligible || artifact.Pruneable
		}
		if eligible {
			plan.Operations = append(plan.Operations, repairOperation{ID: "repository-cleanup", FindingIDs: findingIDs(delivery.Findings, "task-delivery-artifacts"), Mode: "delegated", DelegatedTo: "cleanup", DelegatedPlan: cleanupPlan.ID, Argv: []string{"cleanup", "--project", project, "--apply-safe", cleanupPlan.ID}, Rollback: []string{"use the exact recovery commands and artifact restore identities in the cleanup audit"}, NextAction: "apply only after reviewing every eligible cleanup item"})
		}
	}
	journalPlan, journalErr := eventlog.PlanJournal(w, project, nil, now)
	if journalErr == nil && (journalPlan.DismissCount > 0 || journalPlan.ArchiveCount > 0) {
		plan.Operations = append(plan.Operations, repairOperation{ID: "event-journal", FindingIDs: findingIDs(delivery.Findings, "event-target-missing", "terminal-task-event"), Mode: "delegated", DelegatedTo: "events reconcile", DelegatedPlan: journalPlan.ID, Argv: []string{"events", "reconcile", "--project", project, "--apply-safe", journalPlan.ID}, Rollback: []string{"restore archived files to their original event paths", "append a compensating event; never rewrite original events"}, NextAction: "apply the immutable journal plan"})
	}
	delegated := map[string]bool{}
	for _, operation := range plan.Operations {
		for _, id := range operation.FindingIDs {
			delegated[id] = true
		}
	}
	for _, finding := range delivery.Findings {
		if delegated[finding.ID] {
			continue
		}
		plan.Operations = append(plan.Operations, repairOperation{ID: "manual:" + finding.ID, FindingIDs: []string{finding.ID}, Mode: "manual", Rollback: []string{"no automated mutation is authorized"}, NextAction: finding.NextAction})
	}
	sort.Slice(plan.Preconditions, func(i, j int) bool { return plan.Preconditions[i].FindingID < plan.Preconditions[j].FindingID })
	sort.Slice(plan.Operations, func(i, j int) bool { return plan.Operations[i].ID < plan.Operations[j].ID })
	plan.ID = repairPlanID(plan)
	if observeErr != nil {
		// Unknown external state is preserved in the plan as a manual finding;
		// safe local operations remain previewable, but apply will re-observe and
		// refuse because the immutable finding set cannot become authoritative.
		return plan, observeErr
	}
	return plan, nil
}

func findingIDs(findings []store.DeliveryFinding, classes ...string) []string {
	wanted := map[string]bool{}
	for _, class := range classes {
		wanted[class] = true
	}
	var ids []string
	for _, finding := range findings {
		if wanted[finding.Classification] {
			ids = append(ids, finding.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func repairPlanID(plan repairPlan) string {
	plan.ID = ""
	plan.ObservedAt = time.Time{}
	raw, _ := json.Marshal(plan)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func applyRepairPlan(w *workspace.Workspace, actor, project, requestedID string, now time.Time) (repairAudit, error) {
	audit := repairAudit{}
	err := store.WithFileLock(filepath.Join(w.Root, workspace.Dir, ".reconciliation-repair.lock"), func() error {
		plan, err := planRepairs(w, project, now)
		if err != nil {
			return fmt.Errorf("re-observe repair preconditions: %w", err)
		}
		if requestedID == "" || requestedID != plan.ID || repairPlanID(plan) != plan.ID {
			return fmt.Errorf("reconciliation repair plan is stale or unknown; review a new --dry-run")
		}
		audit = repairAudit{Schema: "reconciliation-repair-audit/v1", PlanID: plan.ID, Project: project, StartedAt: now.UTC(), Operations: []repairOperationAudit{}}
		for _, operation := range plan.Operations {
			state := "manual"
			if operation.Mode == "delegated" {
				state = "pending"
			}
			audit.Operations = append(audit.Operations, repairOperationAudit{ID: operation.ID, State: state, Detail: operation.NextAction})
		}
		if err := writeRepairRecords(w, plan, audit); err != nil {
			return err
		}
		for i, operation := range plan.Operations {
			if operation.Mode != "delegated" {
				continue
			}
			var applyErr error
			switch operation.DelegatedTo {
			case "cleanup":
				_, applyErr = store.ApplyRepositoryCleanup(w, project, operation.DelegatedPlan, time.Now(), w.Root)
			case "events reconcile":
				_, applyErr = eventlog.ApplyJournalPlan(w, actor, project, nil, operation.DelegatedPlan, time.Now())
			default:
				applyErr = fmt.Errorf("unsupported delegated owner %q", operation.DelegatedTo)
			}
			if applyErr != nil {
				audit.Operations[i].State, audit.Operations[i].Detail = "failed", applyErr.Error()
				_ = writeRepairAudit(w, audit)
				return fmt.Errorf("repair operation %s failed; partial audit %s records completed truth: %w", operation.ID, repairAuditPath(w, plan.ID), applyErr)
			}
			audit.Operations[i].State, audit.Operations[i].Detail, audit.Operations[i].CompletedAt = "completed", "delegated owner applied its exact plan", time.Now().UTC()
			if err := writeRepairAudit(w, audit); err != nil {
				return err
			}
		}
		return nil
	})
	return audit, err
}

func writeRepairRecords(w *workspace.Workspace, plan repairPlan, audit repairAudit) error {
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	if err := mdstore.WriteBytes(filepath.Join(w.Root, workspace.Dir, "plans", "reconciliation", plan.ID+".json"), append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return writeRepairAudit(w, audit)
}

func repairAuditPath(w *workspace.Workspace, planID string) string {
	return filepath.Join(w.Root, workspace.Dir, "audits", "reconciliation", planID+".json")
}

func writeRepairAudit(w *workspace.Workspace, audit repairAudit) error {
	raw, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		return err
	}
	return mdstore.WriteBytes(repairAuditPath(w, audit.PlanID), append(raw, '\n'), 0o644)
}
