package eventlog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const JournalPlanSchema = "event-journal-reconciliation/v1"

type JournalItem struct {
	ID             string          `json:"id"`
	Kind           model.EventKind `json:"kind,omitempty"`
	Actor          string          `json:"actor,omitempty"`
	About          string          `json:"about,omitempty"`
	LogicalPath    string          `json:"logical_path"`
	Classification string          `json:"classification"`
	Action         string          `json:"action"`
	Reason         string          `json:"reason"`
	ManualAction   string          `json:"manual_action,omitempty"`
	Bytes          int64           `json:"bytes"`
	Hash           string          `json:"sha256,omitempty"`
}

type JournalPlan struct {
	Schema         string        `json:"schema"`
	Version        int           `json:"version"`
	ID             string        `json:"id"`
	Project        string        `json:"project"`
	ArchiveClasses []string      `json:"archive_classes"`
	ObservedAt     time.Time     `json:"observed_at"`
	Items          []JournalItem `json:"items"`
	TotalBytes     int64         `json:"total_bytes"`
	ArchiveBytes   int64         `json:"archive_bytes"`
	ArchiveCount   int           `json:"archive_count"`
	DismissCount   int           `json:"dismiss_count"`
}

func PlanJournal(w *workspace.Workspace, project string, archiveClasses []string, now time.Time) (JournalPlan, error) {
	classes := normalizeClasses(archiveClasses)
	p := JournalPlan{Schema: JournalPlanSchema, Version: 1, Project: project, ArchiveClasses: classes, ObservedAt: now.UTC(), Items: []JournalItem{}}
	tasks, err := store.ListTasks(w, project, "")
	if err != nil {
		return p, err
	}
	taskByID := map[string]*store.Task{}
	for _, task := range tasks {
		taskByID[task.ID] = task
	}
	events, holes, err := ListReport(w, Query{})
	if err != nil {
		return p, err
	}
	// Terminal dispositions are the append-only resolution mechanism, not
	// compaction candidates. Omitting them also keeps a plan identity stable
	// across its own dismissal phase, which makes crash/restart replay safe.
	allByID := map[string]*Event{}
	for _, event := range events {
		allByID[event.ID] = event
	}
	referenced := map[string]bool{}
	for _, event := range events {
		if event.Kind != model.EventDismissal && allByID[event.About] != nil {
			referenced[event.About] = true
		}
	}
	var scoped []*Event
	for _, event := range events {
		if event.Kind == model.EventDismissal {
			continue
		}
		if taskByID[event.About] != nil || (strings.HasPrefix(event.About, "t-") && !event.Applied && !event.Kind.IsJournal()) {
			scoped = append(scoped, event)
		}
	}
	pendingByTarget := map[string][]*Event{}
	for _, event := range scoped {
		if !event.Applied && !event.Kind.IsJournal() {
			pendingByTarget[event.About] = append(pendingByTarget[event.About], event)
		}
	}
	superseded := map[string]bool{}
	for _, group := range pendingByTarget {
		var proposals []*Event
		for _, event := range group {
			if event.Kind == model.EventProposeStatus {
				proposals = append(proposals, event)
			}
		}
		sort.Slice(proposals, func(i, j int) bool { return proposals[i].ID < proposals[j].ID })
		for _, event := range proposals[:max(0, len(proposals)-1)] {
			superseded[event.ID] = true
		}
	}
	archiveSet := map[string]bool{}
	for _, class := range classes {
		archiveSet[class] = true
	}
	for _, event := range scoped {
		item := classifyJournalItem(w, event, taskByID[event.About], pendingByTarget[event.About], superseded[event.ID], referenced[event.ID], archiveSet)
		p.Items = append(p.Items, item)
		p.TotalBytes += item.Bytes
		if item.Action == "archive" {
			p.ArchiveCount++
			p.ArchiveBytes += item.Bytes
		}
		if item.Action == "dismiss" {
			p.DismissCount++
		}
	}
	for _, path := range holes {
		info, _ := os.Stat(path)
		item := JournalItem{LogicalPath: logicalEventPath(w, path), Classification: "unknown-unreadable", Action: "preserve", Reason: "event record cannot be parsed or verified", ManualAction: "repair the record and rerun; do not delete it"}
		if info != nil {
			item.Bytes = info.Size()
		}
		p.Items = append(p.Items, item)
		p.TotalBytes += item.Bytes
	}
	sort.Slice(p.Items, func(i, j int) bool { return p.Items[i].LogicalPath < p.Items[j].LogicalPath })
	p.ID = journalPlanID(p)
	return p, nil
}

func classifyJournalItem(w *workspace.Workspace, event *Event, task *store.Task, peers []*Event, oldProposal, externallyReferenced bool, archiveSet map[string]bool) JournalItem {
	raw, readErr := os.ReadFile(event.Path)
	item := JournalItem{ID: event.ID, Kind: event.Kind, Actor: event.Actor, About: event.About, LogicalPath: logicalEventPath(w, event.Path), Action: "preserve"}
	if readErr == nil {
		item.Bytes = int64(len(raw))
		sum := sha256.Sum256(raw)
		item.Hash = hex.EncodeToString(sum[:])
	}
	switch {
	case readErr != nil:
		item.Classification, item.Reason, item.ManualAction = "unknown-unreadable", "event bytes became unreadable after parsing", "restore read access and rerun"
	case externallyReferenced:
		item.Classification, item.Reason, item.ManualAction = "externally-referenced", "another durable event references this event", "review the reference chain before disposition"
	case contestedMailbox(event, peers):
		item.Classification, item.Reason, item.ManualAction = "contested", "multiple non-superseding mailbox records disagree on this target", "resolve the competing records explicitly"
	case oldProposal:
		item.Classification, item.Action, item.Reason = "superseded-proposal", "dismiss", "a newer status proposal for the same target supersedes this one"
	case !event.Applied && !event.Kind.IsJournal() && task == nil:
		item.Classification, item.Action, item.Reason = "missing-target", "dismiss", "the mailbox target no longer exists"
	case !event.Applied && !event.Kind.IsJournal() && task.Status == model.StatusDone:
		item.Classification, item.Action, item.Reason = "terminal-target", "dismiss", "the mailbox target is terminal"
	case !event.Applied && !event.Kind.IsJournal() && !event.Dismissed:
		item.Classification, item.Reason, item.ManualAction = "pending-actionable", "mailbox work still has a live target", "apply with sync or explicitly dismiss it"
	case event.Kind.IsJournal():
		item.Classification, item.Reason = "complete-journal", "journal facts are complete when written"
		if archiveSet[item.Classification] {
			item.Action = "archive"
			item.ManualAction = "recover by moving .dacli/events-archive/" + item.LogicalPath + " back to .dacli/events/" + item.LogicalPath
		}
	default:
		item.Classification, item.Reason = "complete-mailbox", "mailbox work already has a terminal disposition"
		if archiveSet[item.Classification] {
			item.Action = "archive"
			item.ManualAction = "recover by moving .dacli/events-archive/" + item.LogicalPath + " back to .dacli/events/" + item.LogicalPath
		}
	}
	return item
}

func contestedMailbox(current *Event, events []*Event) bool {
	if current.Kind != model.EventClaim && current.Kind != model.EventBlock && current.Kind != model.EventDependency {
		return false
	}
	kinds := map[model.EventKind]bool{}
	actors := map[string]bool{}
	for _, event := range events {
		if event.Kind != model.EventClaim && event.Kind != model.EventBlock && event.Kind != model.EventDependency {
			continue
		}
		kinds[event.Kind], actors[event.Actor] = true, true
	}
	return len(kinds) > 1 || (len(events) > 1 && len(actors) > 1 && len(kinds) > 0)
}

func normalizeClasses(in []string) []string {
	if len(in) == 0 {
		in = []string{"complete-journal"}
	}
	seen := map[string]bool{}
	var out []string
	for _, value := range in {
		for _, class := range strings.Split(value, ",") {
			class = strings.TrimSpace(class)
			if (class == "complete-journal" || class == "complete-mailbox") && !seen[class] {
				seen[class] = true
				out = append(out, class)
			}
		}
	}
	sort.Strings(out)
	return out
}

func logicalEventPath(w *workspace.Workspace, path string) string {
	for _, root := range []string{w.EventsDir(), filepath.Join(w.Root, workspace.Dir, "events-archive")} {
		if rel, err := filepath.Rel(root, path); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.Base(path)
}

func journalPlanID(p JournalPlan) string {
	p.ID = ""
	p.ObservedAt = time.Time{}
	raw, _ := json.Marshal(p)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

type JournalSnapshot struct {
	Schema    string        `json:"schema"`
	PlanID    string        `json:"plan_id"`
	Project   string        `json:"project"`
	CreatedAt time.Time     `json:"created_at"`
	Items     []JournalItem `json:"items"`
}

// JournalPhaseHook is nil in production and lets crash/restart fixtures stop
// after each durable phase without weakening the production state machine.
var JournalPhaseHook func(string) error

func ApplyJournalPlan(w *workspace.Workspace, actor, project string, classes []string, requestedID string, now time.Time) (JournalSnapshot, error) {
	var snapshot JournalSnapshot
	err := store.WithFileLock(filepath.Join(w.Root, workspace.Dir, ".event-journal.lock"), func() error {
		plan, err := PlanJournal(w, project, classes, now)
		if err != nil {
			return err
		}
		if requestedID == "" || requestedID != plan.ID || journalPlanID(plan) != plan.ID {
			return fmt.Errorf("event journal plan is stale or unknown; review a new dry-run")
		}
		snapshot = JournalSnapshot{Schema: "event-journal-snapshot/v1", PlanID: plan.ID, Project: project, CreatedAt: now.UTC(), Items: plan.Items}
		snapshotPath := filepath.Join(w.Root, workspace.Dir, "event-journal", plan.ID, "snapshot.json")
		snapshotRaw, readErr := os.ReadFile(snapshotPath)
		switch {
		case readErr == nil:
			var existing JournalSnapshot
			if err := json.Unmarshal(snapshotRaw, &existing); err != nil || existing.PlanID != plan.ID || existing.Schema != "event-journal-snapshot/v1" {
				return fmt.Errorf("existing event journal snapshot cannot be verified")
			}
			snapshot = existing
		case !os.IsNotExist(readErr):
			return readErr
		default:
			raw, err := json.MarshalIndent(snapshot, "", "  ")
			if err != nil {
				return err
			}
			snapshotRaw = append(raw, '\n')
			if err := mdstore.WriteBytes(snapshotPath, snapshotRaw, 0o644); err != nil {
				return err
			}
		}
		snapshotSum := sha256.Sum256(snapshotRaw)
		expectedSidecar := hex.EncodeToString(snapshotSum[:]) + "  snapshot.json\n"
		sidecarPath := snapshotPath + ".sha256"
		sidecarRaw, sidecarErr := os.ReadFile(sidecarPath)
		switch {
		case sidecarErr == nil && string(sidecarRaw) != expectedSidecar:
			return fmt.Errorf("existing event journal snapshot checksum cannot be verified")
		case os.IsNotExist(sidecarErr):
			if err := mdstore.WriteBytes(sidecarPath, []byte(expectedSidecar), 0o644); err != nil {
				return err
			}
		case sidecarErr != nil:
			return sidecarErr
		}
		if JournalPhaseHook != nil {
			if err := JournalPhaseHook("snapshot"); err != nil {
				return err
			}
		}
		for _, item := range plan.Items {
			if item.Action != "dismiss" {
				continue
			}
			event, err := Find(w, item.ID)
			if err != nil {
				return err
			}
			if _, _, err := Dismiss(w, actor, event, "event journal reconciliation "+plan.ID+": "+item.Classification); err != nil {
				return err
			}
		}
		if JournalPhaseHook != nil {
			if err := JournalPhaseHook("dismissal"); err != nil {
				return err
			}
		}
		for _, item := range plan.Items {
			if item.Action != "archive" {
				continue
			}
			src := filepath.Join(w.EventsDir(), filepath.FromSlash(item.LogicalPath))
			dst := filepath.Join(w.Root, workspace.Dir, "events-archive", filepath.FromSlash(item.LogicalPath))
			if _, err := os.Stat(dst); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("inspect archive destination %s: %w", item.ID, err)
			}
			raw, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("read archive candidate %s: %w", item.ID, err)
			}
			sum := sha256.Sum256(raw)
			if hex.EncodeToString(sum[:]) != item.Hash {
				return fmt.Errorf("archive candidate %s changed after snapshot", item.ID)
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			if err := os.Rename(src, dst); err != nil {
				return err
			}
			if JournalPhaseHook != nil {
				if err := JournalPhaseHook("archive"); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return snapshot, err
}
