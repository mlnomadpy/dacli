package reconciliation

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestCommandRendersSameVersionedProjectionAsTextAndJSON(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "reconcile")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = w.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	old := store.ObserveDeliveryPRs
	store.ObserveDeliveryPRs = func(string) ([]store.DeliveryPR, error) { return nil, nil }
	t.Cleanup(func() { store.ObserveDeliveryPRs = old })
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdReconcile(ctx, []string{"--project", "core", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	textOut := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(textOut, repairPlanSchema) || !strings.Contains(textOut, "nothing was written") || !strings.Contains(textOut, "--apply-safe") {
		t.Fatalf("human rendering omitted contract:\n%s", textOut)
	}
	jsonCtx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdReconcile(jsonCtx, []string{"--project", "core"}); err != nil {
		t.Fatal(err)
	}
	jsonOut := jsonCtx.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{`"schema": "` + store.DeliverySchemaVersion + `"`, `"version": 1`, `"findings":`} {
		if !strings.Contains(jsonOut, want) {
			t.Errorf("JSON missing %q:\n%s", want, jsonOut)
		}
	}
}

func TestExplainKeepsCurrentLocalFactsWhenGitHubIsUnknown(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "explain-outage")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = w.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTask(w, "a-root", "core", "Local task", store.TaskOpts{Accept: []string{"visible"}, Estimate: "1,2,3"}); err != nil {
		t.Fatal(err)
	}
	oldObserver := store.ObserveDeliveryPRs
	store.ObserveDeliveryPRs = func(string) ([]store.DeliveryPR, error) { return nil, errors.New("fixture GitHub outage") }
	t.Cleanup(func() { store.ObserveDeliveryPRs = oldObserver })
	oldCache := store.SharedProgressExplainCache
	store.SharedProgressExplainCache = store.NewExplainCache(4, time.Second, time.Minute)
	t.Cleanup(func() { store.SharedProgressExplainCache = oldCache })

	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdExplain(ctx, []string{"--project", "core"}); err != nil {
		t.Fatalf("advisory explain hid local state during GitHub outage: %v", err)
	}
	var got store.ProgressExplain
	if err := json.Unmarshal(ctx.Stdout.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Tasks) != 1 || got.CacheState != "partial-observation" || got.Warning == "" || got.Reconciliation.Reconciled {
		t.Fatalf("partial observation not explicit/useful: %+v", got)
	}
}

func TestExplainTextAndJSONExposeSourcesFreshnessAndRejectedRoles(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "explain-command")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = w.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "core", "Implement command", store.TaskOpts{Accept: []string{"rendered"}, Estimate: "1,2,3"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTask(w, "a-root", "core", "Unrelated command", store.TaskOpts{Accept: []string{"hidden"}, Estimate: "1,2,3"}); err != nil {
		t.Fatal(err)
	}
	for _, role := range []team.Role{{Name: "builder", Kind: "implementer", Grant: "rw", Runtime: "codex-rw", Skills: []string{"go"}}, {Name: "reader", Kind: "reviewer", Grant: "ro", Runtime: "codex", Skills: []string{"review"}}} {
		if err := store.CreateRole(w, "a-root", role); err != nil {
			t.Fatal(err)
		}
	}
	oldObserver := store.ObserveDeliveryPRs
	store.ObserveDeliveryPRs = func(string) ([]store.DeliveryPR, error) { return nil, nil }
	t.Cleanup(func() { store.ObserveDeliveryPRs = oldObserver })
	oldCache := store.SharedProgressExplainCache
	store.SharedProgressExplainCache = store.NewExplainCache(4, time.Second, time.Minute)
	t.Cleanup(func() { store.SharedProgressExplainCache = oldCache })

	textCtx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdExplain(textCtx, []string{"--project", "core"}); err != nil {
		t.Fatal(err)
	}
	textOut := textCtx.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{store.ProgressExplainSchema, "role reader", "rejected:", "source=", "stale=false", "next:"} {
		if !strings.Contains(textOut, want) {
			t.Errorf("text explain missing %q:\n%s", want, textOut)
		}
	}

	jsonCtx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdExplain(jsonCtx, []string{"--project", "core"}); err != nil {
		t.Fatal(err)
	}
	var got store.ProgressExplain
	if err := json.Unmarshal(jsonCtx.Stdout.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatalf("decode explain JSON: %v\n%s", err, jsonCtx.Stdout.(*bytes.Buffer))
	}
	if len(got.Tasks) != 2 || got.Tasks[0].Status.Source == "" || got.Tasks[0].Status.ObservedAt.IsZero() || got.Tasks[0].Status.Stale {
		t.Fatalf("JSON task facts are not sourced/current: %+v", got.Tasks)
	}
	if rejected := got.Tasks[0].RoleRouting.Value.Candidate("reader"); rejected == nil || rejected.Eligible || len(rejected.Exclusions) == 0 {
		t.Fatalf("JSON dropped rejected role: %+v", got.Tasks[0].RoleRouting.Value)
	}
	taskCtx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdExplain(taskCtx, []string{task.ID}); err != nil {
		t.Fatal(err)
	}
	var one store.ProgressExplain
	if err := json.Unmarshal(taskCtx.Stdout.(*bytes.Buffer).Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if len(one.Tasks) != 1 || one.Tasks[0].ID.Value != task.ID {
		t.Fatalf("task-scoped explain = %+v", one.Tasks)
	}
}
