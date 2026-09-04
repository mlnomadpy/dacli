package store

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
)

func TestClaimExpansionIsVersionedReasonedRunBoundAndIdempotent(t *testing.T) {
	w, task, handoff := parentCommitFixture(t, []string{"source.txt"})
	rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(handoff.RunID), "proc.txt"))
	if err != nil || procmon.CompleteRecord(filepath.Join(w.RunDir(handoff.RunID), "proc.txt"), rec, "failed") != nil {
		t.Fatalf("complete prior run: %v", err)
	}
	now := time.Unix(40, 0)
	plan, err := ExpandTaskClaims(w, task, handoff.RunID, "a-root", "review found the verification fixture", []string{"tests/fixture.txt"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Schema != ClaimExpansionSchema || plan.ID == "" || plan.TaskID != task.ID || plan.RunID != handoff.RunID || plan.Actor != "a-root" || plan.Reason == "" || plan.AppliedAt.IsZero() {
		t.Fatalf("claim expansion identity = %+v", plan)
	}
	if !slices.Equal(plan.OldClaims, []string{"source.txt"}) || !slices.Equal(plan.NewClaims, []string{"source.txt", "tests/fixture.txt"}) {
		t.Fatalf("claim expansion old/new = %v -> %v", plan.OldClaims, plan.NewClaims)
	}
	fresh, err := FindTask(w, task.ID)
	if err != nil || !slices.Equal(fresh.Claims(), plan.NewClaims) {
		t.Fatalf("persisted task claims=%v err=%v", fresh.Claims(), err)
	}
	again, err := ExpandTaskClaims(w, fresh, handoff.RunID, "a-root", plan.Reason, []string{"tests/fixture.txt"}, now)
	if err != nil || again.ID != plan.ID {
		t.Fatalf("idempotent expansion=%+v err=%v", again, err)
	}
	if _, err := ExpandTaskClaims(w, fresh, handoff.RunID, "a-root", "different authority", []string{"other"}, now); err == nil || !strings.Contains(err.Error(), "different immutable") {
		t.Fatalf("different expansion reused run: %v", err)
	}
}
