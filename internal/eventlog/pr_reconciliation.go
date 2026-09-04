package eventlog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const RedundantPRPlanSchema = "redundant-pr-event-reconciliation/v1"

type MergedPRObservation struct {
	Number      int
	URL         string
	Head        string
	HeadOID     string
	MergeCommit string
	Source      string
}

type RedundantPRItem struct {
	EventID        string    `json:"event_id"`
	TaskID         string    `json:"task_id"`
	TaskGeneration int       `json:"task_generation"`
	PRNumber       int       `json:"pr_number"`
	PRURL          string    `json:"pr_url"`
	Head           string    `json:"head"`
	HeadOID        string    `json:"head_oid"`
	MergeCommit    string    `json:"merge_commit"`
	ObservedAt     time.Time `json:"observed_at"`
	ObservedSource string    `json:"observed_source"`
	Reason         string    `json:"dismissal_reason"`
}

type RedundantPRPlan struct {
	Schema     string            `json:"schema"`
	Version    int               `json:"version"`
	ID         string            `json:"id"`
	Project    string            `json:"project"`
	ObservedAt time.Time         `json:"observed_at"`
	Items      []RedundantPRItem `json:"items"`
}

func PlanRedundantPREvents(w *workspace.Workspace, project string, prs []MergedPRObservation, now time.Time) (RedundantPRPlan, error) {
	plan := RedundantPRPlan{Schema: RedundantPRPlanSchema, Version: 1, Project: project, ObservedAt: now.UTC(), Items: []RedundantPRItem{}}
	tasks, err := store.ListTasks(w, project, "")
	if err != nil {
		return plan, err
	}
	byID := make(map[string]*store.Task, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}
	events, holes, err := ListReport(w, Query{Pending: true})
	if err != nil {
		return plan, err
	}
	if len(holes) > 0 {
		return plan, fmt.Errorf("cannot reconcile PR events while %d event record(s) are unreadable", len(holes))
	}
	byURL := map[string][]MergedPRObservation{}
	for _, pr := range prs {
		if pr.URL != "" && pr.MergeCommit != "" && pr.Head != "" && pr.HeadOID != "" {
			byURL[pr.URL] = append(byURL[pr.URL], pr)
		}
	}
	for _, event := range events {
		task := byID[event.About]
		url, ok := prOpenedURL(event)
		if !ok || task == nil || task.Status != model.StatusDone || task.Generation() != 0 || !taskAcceptanceComplete(task) {
			continue
		}
		matches := byURL[url]
		if len(matches) != 1 || matches[0].Head != store.TaskBranch(task) {
			continue
		}
		pr := matches[0]
		plan.Items = append(plan.Items, RedundantPRItem{
			EventID: event.ID, TaskID: task.ID, TaskGeneration: task.Generation(),
			PRNumber: pr.Number, PRURL: pr.URL, Head: pr.Head, HeadOID: pr.HeadOID,
			MergeCommit: pr.MergeCommit, ObservedAt: now.UTC(), ObservedSource: pr.Source,
			Reason: "redundant PR-open event for an accepted task and its observably merged canonical generation",
		})
	}
	sort.Slice(plan.Items, func(i, j int) bool { return plan.Items[i].EventID < plan.Items[j].EventID })
	plan.ID = redundantPRPlanID(plan)
	return plan, nil
}

func ApplyRedundantPRPlan(w *workspace.Workspace, actor string, plan RedundantPRPlan) error {
	if plan.ID == "" || redundantPRPlanID(plan) != plan.ID {
		return fmt.Errorf("redundant PR event plan is invalid or stale")
	}
	return store.WithFileLock(filepath.Join(w.Root, workspace.Dir, ".event-pr-reconciliation.lock"), func() error {
		for _, item := range plan.Items {
			event, err := Find(w, item.EventID)
			if err != nil {
				return err
			}
			task, err := store.FindTask(w, item.TaskID)
			if err != nil {
				return err
			}
			url, exactPREvent := prOpenedURL(event)
			if !exactPREvent || event.About != task.ID || url != item.PRURL || task.Status != model.StatusDone || task.Generation() != item.TaskGeneration || !taskAcceptanceComplete(task) || store.TaskBranch(task) != item.Head || item.PRNumber <= 0 || item.HeadOID == "" || item.MergeCommit == "" || item.ObservedAt.IsZero() || item.ObservedSource == "" {
				return fmt.Errorf("redundant PR event plan became stale for event %s", item.EventID)
			}
		}
		dir := filepath.Join(w.Root, workspace.Dir, "event-pr-reconciliation", plan.ID)
		raw, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return err
		}
		if err := mdstore.WriteBytes(filepath.Join(dir, "snapshot.json"), append(raw, '\n'), 0o644); err != nil {
			return err
		}
		for _, item := range plan.Items {
			event, err := Find(w, item.EventID)
			if err != nil {
				return err
			}
			reason := fmt.Sprintf("%s; plan=%s task=%s generation=%d pr=%s merge_commit=%s observed_at=%s source=%s", item.Reason, plan.ID, item.TaskID, item.TaskGeneration, item.PRURL, item.MergeCommit, item.ObservedAt.Format(time.RFC3339), item.ObservedSource)
			if _, _, err := Dismiss(w, actor, event, reason); err != nil {
				return err
			}
		}
		return nil
	})
}

func redundantPRPlanID(plan RedundantPRPlan) string {
	canonical := plan
	canonical.Items = append([]RedundantPRItem(nil), plan.Items...)
	canonical.ID = ""
	canonical.ObservedAt = time.Time{}
	for i := range canonical.Items {
		canonical.Items[i].ObservedAt = time.Time{}
	}
	raw, _ := json.Marshal(canonical)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func prOpenedURL(event *Event) (string, bool) {
	if event.Kind != model.EventComment {
		return "", false
	}
	const prefix = "PR opened: "
	body := strings.TrimSpace(event.Body)
	url := strings.TrimPrefix(body, prefix)
	return url, url != body && strings.HasPrefix(url, "https://") && !strings.Contains(url, " ")
}

func taskAcceptanceComplete(task *store.Task) bool {
	boxes := task.Acceptance()
	if len(boxes) == 0 {
		return false
	}
	for _, box := range boxes {
		if !box.Done {
			return false
		}
	}
	return true
}
