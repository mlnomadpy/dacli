package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const cyclePhaseSchema = "loop-phase-journal/v1"

type cyclePhase string

const (
	phaseSpawned   cyclePhase = "spawned"
	phaseWaited    cyclePhase = "waited"
	phaseVerified  cyclePhase = "verified"
	phasePushed    cyclePhase = "pushed"
	phasePRCreated cyclePhase = "pr-created"
	phaseCIPending cyclePhase = "ci-pending"
	phaseMerged    cyclePhase = "merged"
	phaseAccepted  cyclePhase = "record-accepted"
)

var phaseOrder = []cyclePhase{phaseSpawned, phaseWaited, phaseVerified, phasePushed, phasePRCreated, phaseCIPending, phaseMerged, phaseAccepted}

type taskPhaseCheckpoint struct {
	TaskID     string     `json:"task_id"`
	Sequence   int        `json:"sequence"`
	Generation int        `json:"generation"`
	Branch     string     `json:"branch"`
	RunID      string     `json:"run_id,omitempty"`
	Phase      cyclePhase `json:"phase"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type cyclePhaseJournal struct {
	Schema  string                `json:"schema"`
	Version int                   `json:"version"`
	Project string                `json:"project"`
	Cycle   int                   `json:"cycle"`
	Tasks   []taskPhaseCheckpoint `json:"tasks"`
}

func phaseJournalFile(w *workspace.Workspace, project string) string {
	return filepath.Join(w.Root, workspace.Dir, "loop", project+"-phases.json")
}

func writePhaseJournal(w *workspace.Workspace, journal cyclePhaseJournal) error {
	journal.Schema, journal.Version = cyclePhaseSchema, 1
	slices.SortFunc(journal.Tasks, func(a, b taskPhaseCheckpoint) int { return a.Sequence - b.Sequence })
	raw, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	path := phaseJournalFile(w, journal.Project)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := writeStateFile(path, string(append(raw, '\n'))); err != nil {
		return err
	}
	got, err := readPhaseJournal(w, journal.Project)
	if err != nil {
		return fmt.Errorf("validate persisted phase journal: %w", err)
	}
	if !slices.Equal(got.Tasks, journal.Tasks) || got.Cycle != journal.Cycle {
		return fmt.Errorf("validate persisted phase journal: got %+v, want %+v", got, journal)
	}
	return nil
}

func readPhaseJournal(w *workspace.Workspace, project string) (cyclePhaseJournal, error) {
	journal := cyclePhaseJournal{Schema: cyclePhaseSchema, Version: 1, Project: project}
	raw, err := os.ReadFile(phaseJournalFile(w, project))
	if os.IsNotExist(err) {
		return journal, nil
	}
	if err != nil {
		return journal, err
	}
	if err := json.Unmarshal(raw, &journal); err != nil {
		return journal, fmt.Errorf("decode phase journal: %w", err)
	}
	if journal.Schema != cyclePhaseSchema || journal.Version != 1 || journal.Project != project {
		return journal, fmt.Errorf("invalid phase journal for project %s", project)
	}
	for _, checkpoint := range journal.Tasks {
		if checkpoint.TaskID == "" || checkpoint.Sequence <= 0 || checkpoint.Branch == "" || phaseIndex(checkpoint.Phase) < 0 {
			return journal, fmt.Errorf("invalid phase checkpoint for project %s: %+v", project, checkpoint)
		}
	}
	return journal, nil
}

func phaseIndex(phase cyclePhase) int { return slices.Index(phaseOrder, phase) }

func phaseAtLeast(got, want cyclePhase) bool {
	return phaseIndex(got) >= phaseIndex(want)
}

func (d *driver) taskPhase(t *store.Task) (taskPhaseCheckpoint, bool) {
	for _, checkpoint := range d.phases.Tasks {
		if checkpoint.Sequence == t.Seq && checkpoint.Generation == t.Generation() {
			return checkpoint, true
		}
	}
	return taskPhaseCheckpoint{}, false
}

// checkpointTaskPhase is a durable post-operation boundary between externally
// visible loop operations. A later phase is never moved backwards, and success
// is not returned until the atomic file has been re-read. This makes a killed
// process replay observation/idempotent reconciliation, not spawn or acceptance.
func (d *driver) checkpointTaskPhase(t *store.Task, phase cyclePhase) bool {
	if d.cfg.dryRun {
		return true
	}
	runID := latestTaskRunID(d.w, t.ID)
	checkpoint := taskPhaseCheckpoint{
		TaskID: t.ID, Sequence: t.Seq, Generation: t.Generation(), Branch: taskBranch(t),
		RunID: runID, Phase: phase, UpdatedAt: d.now().UTC(),
	}
	found := false
	for i := range d.phases.Tasks {
		current := &d.phases.Tasks[i]
		if current.Sequence != t.Seq || current.Generation != t.Generation() {
			continue
		}
		found = true
		if phaseAtLeast(current.Phase, phase) {
			return true
		}
		if checkpoint.RunID == "" {
			checkpoint.RunID = current.RunID
		}
		*current = checkpoint
		break
	}
	if !found {
		d.phases.Tasks = append(d.phases.Tasks, checkpoint)
	}
	d.phases.Project = d.cfg.project
	d.phases.Cycle = d.gov.Cycle() + 1
	if err := writePhaseJournal(d.w, d.phases); err != nil {
		d.phaseErr = fmt.Errorf("persist %s checkpoint for %s: %w", phase, t.ID, err)
		return false
	}
	return true
}

// latestTaskRunID derives provenance from each run's durable proc record, not
// from global run order. A width>1 wait necessarily ends with one globally
// newest run; assigning that ID to every task made exact recovery refs lie.
func latestTaskRunID(w *workspace.Workspace, taskID string) string {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return ""
	}
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if !entry.IsDir() {
			continue
		}
		rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(entry.Name()), "proc.txt"))
		if err == nil && rec.RunID == entry.Name() && rec.Task == taskID {
			return entry.Name()
		}
	}
	return ""
}

func (d *driver) clearTaskPhase(t *store.Task) bool {
	if d.cfg.dryRun {
		return true
	}
	kept := d.phases.Tasks[:0]
	for _, checkpoint := range d.phases.Tasks {
		if checkpoint.Sequence != t.Seq || checkpoint.Generation != t.Generation() {
			kept = append(kept, checkpoint)
		}
	}
	d.phases.Tasks = kept
	if err := writePhaseJournal(d.w, d.phases); err != nil {
		d.phaseErr = fmt.Errorf("clear abandoned phase checkpoint for %s: %w", t.ID, err)
		return false
	}
	return true
}

func (d *driver) restoreLandingPhases() {
	for _, checkpoint := range d.phases.Tasks {
		if !phaseAtLeast(checkpoint.Phase, phasePRCreated) || phaseAtLeast(checkpoint.Phase, phaseAccepted) {
			continue
		}
		if !hasPendingAccept(d.pendingAccept, checkpoint.Sequence, checkpoint.Generation) {
			d.pendingAccept = append(d.pendingAccept, pendingAccept{Seq: checkpoint.Sequence, Branch: checkpoint.Branch, Generation: checkpoint.Generation, GenerationSet: true})
		}
		if !phaseAtLeast(checkpoint.Phase, phaseMerged) && !slices.Contains(d.pendingLand, checkpoint.Branch) {
			d.pendingLand = append(d.pendingLand, checkpoint.Branch)
		}
	}
}
