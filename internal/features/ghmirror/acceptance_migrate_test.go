package ghmirror

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func mappedAcceptanceTask(t *testing.T, w *workspace.Workspace, body string) *store.Task {
	t.Helper()
	task, err := store.CreateTask(w, "a-root", "core", "historical adopted task", store.TaskOpts{Accept: []string{"local criterion"}})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	task.Doc.SetSection("Acceptance", "- [x] local criterion\n")
	task.Doc.Front.SetBlock("github", githubBlock(42, "owner/repo"))
	if err := store.SaveTask(task); err != nil {
		t.Fatalf("save task mapping: %v", err)
	}
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "issue" && args[1] == "view" {
			return fmt.Sprintf(`{"number":42,"title":"historical adopted task","body":%q,"state":"open"}`, body), nil
		}
		return "", fmt.Errorf("unexpected gh call: %v", args)
	}
	return task
}

func migrationPlanID(t *testing.T, output string) string {
	t.Helper()
	fields := strings.Fields(output)
	if len(fields) < 4 || fields[0] != "acceptance" || fields[1] != "migration" || fields[2] != "plan" || len(fields[3]) != 64 {
		t.Fatalf("cannot parse migration plan id from:\n%s", output)
	}
	return fields[3]
}

func TestTaskAcceptanceMigratePreviewApplyAndReplay(t *testing.T) {
	w := mirrorWorkspace(t)
	body := "Background.\n\n## Acceptance criteria\n- [x] local criterion\n- [X] imported result\n"
	task := mappedAcceptanceTask(t, w, body)
	before, err := os.ReadFile(task.Path)
	if err != nil {
		t.Fatalf("read task before preview: %v", err)
	}
	ctx, out := releaseCtx(t, w)
	if err := cmdTaskAcceptanceMigrate(ctx, []string{task.ID, "--dry-run"}); err != nil {
		t.Fatalf("preview: %v\n%s", err, out.String())
	}
	planID := migrationPlanID(t, out.String())
	for _, want := range []string{`acceptance criterion: "local criterion" (unchecked)`, `acceptance criterion: "imported result" (unchecked)`, "nothing was written", "--apply " + planID} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("preview omitted %q:\n%s", want, out.String())
		}
	}
	afterPreview, _ := os.ReadFile(task.Path)
	if string(afterPreview) != string(before) {
		t.Fatal("preview changed the task")
	}
	planPath := filepath.Join(w.Root, workspace.Dir, "plans", "acceptance", planID+".json")
	if _, err := os.Stat(planPath); !os.IsNotExist(err) {
		t.Fatalf("preview persisted plan %s: %v", planPath, err)
	}

	out.Reset()
	if err := cmdTaskAcceptanceMigrate(ctx, []string{task.ID, "--apply", planID}); err != nil {
		t.Fatalf("apply: %v\n%s", err, out.String())
	}
	reloaded, err := store.FindTask(w, task.ID)
	if err != nil {
		t.Fatalf("reload migrated task: %v", err)
	}
	boxes := reloaded.Acceptance()
	if len(boxes) != 2 || !boxes[0].Done || boxes[1].Done || boxes[1].Text != "imported result" {
		t.Fatalf("migrated acceptance = %#v; local state must survive and remote state must remain unchecked", boxes)
	}
	record, ok := reloaded.Doc.Front.GetBlock("github_acceptance_migration")
	if !ok {
		t.Fatal("migration audit record missing")
	}
	for _, want := range []string{"plan: " + planID, "issue: 42", "repo: owner/repo", "body_digest: sha256:", "actor: a-root", "applied_at:"} {
		if !strings.Contains(record, want) {
			t.Fatalf("migration record omitted %q:\n%s", want, record)
		}
	}
	planRaw, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read persisted plan: %v", err)
	}
	var persisted acceptanceMigrationPlan
	if err := json.Unmarshal(planRaw, &persisted); err != nil {
		t.Fatalf("decode persisted plan: %v", err)
	}
	if persisted.ID != planID || persisted.TaskID != task.ID || persisted.Issue != 42 || persisted.BodyDigest != extractIssueAcceptance(body).BodyDigest {
		t.Fatalf("persisted plan = %#v", persisted)
	}

	taskAfterApply, _ := os.ReadFile(reloaded.Path)
	planAfterApply := append([]byte(nil), planRaw...)
	out.Reset()
	if err := cmdTaskAcceptanceMigrate(ctx, []string{task.ID, "--apply", planID}); err != nil {
		t.Fatalf("replay: %v\n%s", err, out.String())
	}
	taskAfterReplay, _ := os.ReadFile(reloaded.Path)
	planAfterReplay, _ := os.ReadFile(planPath)
	if string(taskAfterReplay) != string(taskAfterApply) || string(planAfterReplay) != string(planAfterApply) {
		t.Fatal("replaying an applied immutable plan changed persisted state")
	}
}

func TestTaskAcceptanceMigrateRefusesStalePlan(t *testing.T) {
	w := mirrorWorkspace(t)
	task := mappedAcceptanceTask(t, w, "## Acceptance criteria\n- original criterion\n")
	ctx, out := releaseCtx(t, w)
	if err := cmdTaskAcceptanceMigrate(ctx, []string{task.ID, "--dry-run"}); err != nil {
		t.Fatalf("preview: %v", err)
	}
	staleID := migrationPlanID(t, out.String())
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		return `{"number":42,"title":"historical adopted task","body":"## Acceptance criteria\n- changed criterion","state":"open"}`, nil
	}
	before, _ := os.ReadFile(task.Path)
	out.Reset()
	err := cmdTaskAcceptanceMigrate(ctx, []string{task.ID, "--apply", staleID})
	if err == nil || !strings.Contains(err.Error(), "plan changed") {
		t.Fatalf("stale apply error = %v, want plan-changed refusal", err)
	}
	after, _ := os.ReadFile(task.Path)
	if string(after) != string(before) {
		t.Fatal("stale plan refusal changed the task")
	}
	stalePath := filepath.Join(w.Root, workspace.Dir, "plans", "acceptance", staleID+".json")
	if _, statErr := os.Stat(stalePath); !os.IsNotExist(statErr) {
		t.Fatalf("stale plan was persisted: %v", statErr)
	}
}

func TestTaskAcceptanceMigrateFromSectionIsExact(t *testing.T) {
	w := mirrorWorkspace(t)
	body := "- [ ] unrelated checklist\n\n## Acceptance\n- wrong section\n\n## Acceptance criteria\n- selected criterion\n"
	task := mappedAcceptanceTask(t, w, body)
	ctx, out := releaseCtx(t, w)
	if err := cmdTaskAcceptanceMigrate(ctx, []string{task.ID, "--from-section", "Acceptance criteria", "--dry-run"}); err != nil {
		t.Fatalf("section preview: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `acceptance criterion: "selected criterion"`) {
		t.Fatalf("selected section missing:\n%s", out.String())
	}
	for _, excluded := range []string{"unrelated checklist", "wrong section"} {
		if strings.Contains(out.String(), `acceptance criterion: "`+excluded+`"`) {
			t.Fatalf("preview imported %q outside selected section:\n%s", excluded, out.String())
		}
	}
}
