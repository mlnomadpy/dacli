package reconciliation

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func repairFixture(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "repair")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.email", "test@example.test"}, {"config", "user.name", "test"}, {"commit", "--allow-empty", "-qm", "base"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = w.Root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	old := store.ObserveDeliveryPRs
	store.ObserveDeliveryPRs = func(string) ([]store.DeliveryPR, error) { return nil, nil }
	t.Cleanup(func() { store.ObserveDeliveryPRs = old })
	return w
}

func TestRepairPlanDelegatesToOwnersAndRefusesStaleEvidence(t *testing.T) {
	w := repairFixture(t)
	if _, err := eventlog.Append(w, "a-root", model.EventComment, "t-missing-one", "", "obsolete"); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	plan, err := planRepairs(w, "core", at)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Schema != repairPlanSchema || len(plan.Preconditions) == 0 || plan.ID != repairPlanID(plan) {
		t.Fatalf("repair plan is not immutable/sourced: %+v", plan)
	}
	foundDelegation := false
	for _, operation := range plan.Operations {
		if operation.DelegatedTo == "events reconcile" {
			foundDelegation = operation.DelegatedPlan != "" && len(operation.Argv) > 0 && len(operation.Rollback) > 0
		}
	}
	if !foundDelegation {
		t.Fatalf("missing event-journal delegation: %+v", plan.Operations)
	}
	if _, err := eventlog.Append(w, "a-root", model.EventComment, "t-missing-two", "", "new evidence"); err != nil {
		t.Fatal(err)
	}
	if _, err := applyRepairPlan(w, "a-root", "core", plan.ID, at.Add(time.Second)); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale repair plan applied: %v", err)
	}
	if _, err := os.Stat(repairAuditPath(w, plan.ID)); !os.IsNotExist(err) {
		t.Fatalf("stale refusal wrote an audit or mutation: %v", err)
	}

	fresh, err := planRepairs(w, "core", at.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	audit, err := applyRepairPlan(w, "a-root", "core", fresh.ID, at.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	completed := false
	for _, operation := range audit.Operations {
		completed = completed || operation.ID == "event-journal" && operation.State == "completed"
	}
	if !completed {
		t.Fatalf("delegated operation truth missing: %+v", audit)
	}
	if _, err := os.Stat(filepath.Join(w.Root, workspace.Dir, "plans", "reconciliation", fresh.ID+".json")); err != nil {
		t.Fatalf("immutable plan was not persisted: %v", err)
	}
}

func TestRepairPlanKeepsUnknownAndUnsupportedFindingsManual(t *testing.T) {
	w := repairFixture(t)
	store.ObserveDeliveryPRs = func(string) ([]store.DeliveryPR, error) { return nil, errors.New("GitHub unavailable") }
	plan, err := planRepairs(w, "core", time.Now())
	if err == nil {
		t.Fatal("unknown GitHub state was treated as authoritative")
	}
	found := false
	for _, operation := range plan.Operations {
		if strings.Contains(operation.ID, "github-state-unknown") {
			found = operation.Mode == "manual" && operation.DelegatedTo == "" && strings.TrimSpace(operation.NextAction) != ""
		}
	}
	if !found {
		t.Fatalf("unsupported unknown finding was not explicit/manual: %+v", plan.Operations)
	}
}
