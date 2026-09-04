package store

// Issue #1021: claim growth is authority, not something learned from a
// provider's attempted writes. Persist the exact old/new scope and reason
// before changing the task so a relaunch can only consume an audited decision.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const (
	ClaimExpansionSchema = "claim-expansion/v1"
	ClaimExpansionFile   = "claim-expansion-v1.json"
)

type ClaimExpansion struct {
	Schema    string    `json:"schema"`
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	RunID     string    `json:"run_id"`
	Actor     string    `json:"actor"`
	Reason    string    `json:"reason"`
	OldClaims []string  `json:"old_claims"`
	NewClaims []string  `json:"new_claims"`
	CreatedAt time.Time `json:"created_at"`
	AppliedAt time.Time `json:"applied_at,omitempty"`
}

func ClaimExpansionPath(w *workspace.Workspace, runID string) string {
	return filepath.Join(w.RunDir(runID), ClaimExpansionFile)
}

func claimExpansionID(plan ClaimExpansion) (string, error) {
	plan.ID = ""
	plan.AppliedAt = time.Time{}
	raw, err := json.Marshal(plan)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func normalizeClaimList(claims []string) ([]string, error) {
	var out []string
	for _, raw := range claims {
		claim := filepath.ToSlash(filepath.Clean(strings.TrimSpace(raw)))
		if claim == "" || claim == "." || claim == ".." || strings.HasPrefix(claim, "../") || filepath.IsAbs(raw) || claim == workspace.Dir || strings.HasPrefix(claim, workspace.Dir+"/") {
			return nil, fmt.Errorf("invalid repository-relative claim %q", raw)
		}
		out = append(out, strings.Trim(claim, "/"))
	}
	slices.Sort(out)
	out = slices.Compact(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("claim expansion requires at least one exact path")
	}
	return out, nil
}

func loadClaimExpansion(w *workspace.Workspace, runID string) (ClaimExpansion, error) {
	var plan ClaimExpansion
	raw, err := os.ReadFile(ClaimExpansionPath(w, runID))
	if err != nil {
		return plan, err
	}
	if err := json.Unmarshal(raw, &plan); err != nil {
		return plan, err
	}
	want, err := claimExpansionID(plan)
	if err != nil || plan.Schema != ClaimExpansionSchema || plan.ID != want || plan.TaskID == "" || plan.RunID != runID || plan.Actor == "" || plan.Reason == "" || len(plan.NewClaims) == 0 || plan.CreatedAt.IsZero() {
		return plan, fmt.Errorf("invalid %s", ClaimExpansionSchema)
	}
	return plan, nil
}

func writeClaimExpansion(w *workspace.Workspace, plan ClaimExpansion) error {
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return writeRootHandoffAtomic(ClaimExpansionPath(w, plan.RunID), append(raw, '\n'), 0o600)
}

// ExpandTaskClaims applies one owner-authorized expansion. The immutable plan
// is written before the task file; retries complete a crash between those two
// writes, and a different plan may never overwrite the run's authority record.
func ExpandTaskClaims(w *workspace.Workspace, task *Task, runID, actor, reason string, additions []string, now time.Time) (ClaimExpansion, error) {
	var plan ClaimExpansion
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return plan, fmt.Errorf("claim expansion requires a reason")
	}
	rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(runID), "proc.txt"))
	if err != nil || rec.RunID != runID || rec.Task != task.ID {
		return plan, fmt.Errorf("claim expansion run %s does not belong to task %s", runID, task.ID)
	}
	if strings.TrimSpace(rec.Outcome) == "" {
		return plan, fmt.Errorf("claim expansion refuses while run %s is still live; expansion applies only to a later relaunch", runID)
	}
	add, err := normalizeClaimList(additions)
	if err != nil {
		return plan, err
	}
	if existing, loadErr := loadClaimExpansion(w, runID); loadErr == nil {
		current := append([]string(nil), task.Claims()...)
		slices.Sort(current)
		allRecorded := true
		for _, claim := range add {
			allRecorded = allRecorded && slices.Contains(existing.NewClaims, claim)
		}
		if existing.Actor == actor && existing.Reason == reason && slices.Equal(current, existing.NewClaims) && allRecorded {
			return existing, nil
		}
		return ClaimExpansion{}, fmt.Errorf("run %s already has a different immutable claim expansion", runID)
	} else if !errors.Is(loadErr, fs.ErrNotExist) {
		return ClaimExpansion{}, loadErr
	}
	var old []string
	if len(task.Claims()) > 0 {
		old, err = normalizeClaimList(task.Claims())
		if err != nil {
			return plan, fmt.Errorf("current task claims: %w", err)
		}
	}
	newClaims := append(append([]string(nil), old...), add...)
	slices.Sort(newClaims)
	newClaims = slices.Compact(newClaims)
	if slices.Equal(old, newClaims) {
		return plan, fmt.Errorf("claim expansion adds no new scope")
	}
	plan = ClaimExpansion{Schema: ClaimExpansionSchema, TaskID: task.ID, RunID: runID, Actor: actor, Reason: reason, OldClaims: old, NewClaims: newClaims, CreatedAt: now.UTC()}
	plan.ID, err = claimExpansionID(plan)
	if err != nil {
		return ClaimExpansion{}, err
	}
	if err := writeClaimExpansion(w, plan); err != nil {
		return ClaimExpansion{}, err
	}
	err = WithTask(w, task, func(fresh *Task) error {
		var current []string
		if len(fresh.Claims()) > 0 {
			var currentErr error
			current, currentErr = normalizeClaimList(fresh.Claims())
			if currentErr != nil {
				return currentErr
			}
		}
		if !slices.Equal(current, plan.OldClaims) && !slices.Equal(current, plan.NewClaims) {
			return fmt.Errorf("task claim scope changed since expansion plan: current=%v old=%v new=%v", current, plan.OldClaims, plan.NewClaims)
		}
		if slices.Equal(current, plan.OldClaims) {
			fresh.Doc.Front.SetList("claims", plan.NewClaims)
			if err := SaveTask(fresh); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ClaimExpansion{}, err
	}
	if plan.AppliedAt.IsZero() {
		plan.AppliedAt = now.UTC()
		if err := writeClaimExpansion(w, plan); err != nil {
			return ClaimExpansion{}, err
		}
	}
	return plan, nil
}
