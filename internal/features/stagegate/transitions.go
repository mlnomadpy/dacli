package stagegate

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

type stageReceipt struct{ Key, Actor, Outcome, Reason, State, BeforeStage, AfterStage string }

var stageReceiptWrite = writeStageReceipt

func stageTransitionPath(dir, key string) string {
	return filepath.Join(dir, fmt.Sprintf("%x.json", sha256.Sum256([]byte(key))))
}

func applyStageTransition(w *workspace.Workspace, actor, slug, projectID, key, before, outcome, reason string) (string, []gates.Check, bool, error) {
	projectDir := filepath.Join(w.ProjectsDir(), slug)
	receipts := filepath.Join(projectDir, "stage.transitions")
	var newStage string
	var unmet []gates.Check
	var replay bool
	err := store.WithFileLock(filepath.Join(projectDir, "stage.transition.lock"), func() error {
		var err error
		newStage, unmet, replay, err = applyLockedStageTransition(w, actor, slug, projectID, key, before, outcome, reason, projectDir, receipts)
		return err
	})
	return newStage, unmet, replay, err
}

func applyLockedStageTransition(w *workspace.Workspace, actor, slug, projectID, key, before, outcome, reason, projectDir, receipts string) (string, []gates.Check, bool, error) {
	p, err := store.LoadProject(w, slug)
	if err != nil {
		return "", nil, false, err
	}
	current, _ := p.Doc.Front.Get("template_stage")
	if current == "" {
		current = "solo"
	}
	path := stageTransitionPath(receipts, key)
	var newStage string
	var unmet []gates.Check
	r, exists, err := readStageReceipt(path)
	if err != nil {
		return "", nil, false, err
	}
	if !exists {
		r = stageReceipt{Key: key, Actor: actor, Outcome: outcome, Reason: reason, State: "pending", BeforeStage: current}
		if outcome == "success" {
			status, statusErr := gates.Status(w, slug)
			if statusErr != nil {
				return "", nil, false, statusErr
			}
			for _, check := range status.Checks {
				if !check.OK {
					unmet = append(unmet, check)
				}
			}
			if len(unmet) > 0 {
				return "", unmet, false, nil
			}
			r.AfterStage = "complete"
			if status.Next != nil {
				r.AfterStage = status.Next.Name
			}
		}
		if err := stageReceiptWrite(path, r); err != nil {
			return "", nil, false, err
		}
	} else if r.Actor != actor || r.Outcome != outcome || r.Reason != reason {
		return "", nil, false, fmt.Errorf("transition key %q was already used with different attributes", key)
	}
	replay := r.State == "applied"
	if r.State == "pending" {
		if outcome == "success" {
			if current == r.BeforeStage {
				newStage, unmet, err = gates.Advance(w, slug)
				if err != nil || len(unmet) > 0 {
					return newStage, unmet, false, err
				}
				if newStage != r.AfterStage {
					return "", nil, false, fmt.Errorf("transition %q advanced to %q, expected %q", key, newStage, r.AfterStage)
				}
			} else if r.AfterStage != "" && current == r.AfterStage {
				newStage = current
			} else {
				return "", nil, false, fmt.Errorf("cannot reconcile transition %q: stage is %q", key, current)
			}
		} else {
			newStage = current
		}
		r.State = "applied"
		if err := stageReceiptWrite(path, r); err != nil {
			return "", nil, false, err
		}
	} else {
		newStage = r.AfterStage
		if newStage == "" {
			newStage = current
		}
	}
	if outcome == "terminal" {
		deadPath := stageTransitionPath(filepath.Join(projectDir, "stage.dead-letter"), key)
		if _, err := os.Stat(deadPath); os.IsNotExist(err) {
			if err := stageReceiptWrite(deadPath, r); err != nil {
				return "", nil, false, err
			}
		} else if err != nil {
			return "", nil, false, err
		}
	}
	body := fmt.Sprintf("stage transition key=%q outcome=%s stage=%q", key, outcome, before)
	if reason != "" {
		body += fmt.Sprintf(" reason=%q", reason)
	}
	if err := ensureStageAudit(w, actor, projectID, body); err != nil {
		return "", nil, false, err
	}
	return newStage, nil, replay, nil
}

func readStageReceipt(path string) (stageReceipt, bool, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return stageReceipt{}, false, nil
	}
	if err != nil {
		return stageReceipt{}, false, err
	}
	var r stageReceipt
	err = json.Unmarshal(b, &r)
	return r, true, err
}
func writeStageReceipt(path string, r stageReceipt) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return mdstore.WriteBytes(path, b, 0o600)
}
func ensureStageAudit(w *workspace.Workspace, actor, about, body string) error {
	events, err := eventlog.List(w, eventlog.Query{About: about, Kinds: []model.EventKind{model.EventRun}})
	if err != nil {
		return err
	}
	for _, e := range events {
		if e.Actor == actor && strings.TrimSpace(e.Body) == body {
			return nil
		}
	}
	_, err = eventlog.Append(w, actor, model.EventRun, about, "agent", body)
	return err
}
