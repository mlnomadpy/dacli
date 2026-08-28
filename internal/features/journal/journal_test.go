package journal

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func journalWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	if value, ok := os.LookupEnv(agentid.EnvVar); ok {
		t.Setenv(agentid.EnvVar, value)
		_ = os.Unsetenv(agentid.EnvVar)
	}
	w, err := workspace.Init(t.TempDir(), "journal")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, agentid.RootID, "core", "target", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eventlog.Append(w, "a-worker", model.EventCommit, task.ID, "", "sha abc"); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestCommandDryRunTextAndJSONSharePlanWithoutWriting(t *testing.T) {
	w := journalWorkspace(t)
	before := countFiles(t, filepath.Join(w.Root, workspace.Dir))
	text := &bytes.Buffer{}
	if err := cmdReconcile(&clikit.Ctx{Cwd: w.Root, Stdout: text, Stderr: &bytes.Buffer{}}, []string{"--project", "core", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	after := countFiles(t, filepath.Join(w.Root, workspace.Dir))
	if before != after || !strings.Contains(text.String(), eventlog.JournalPlanSchema) || !strings.Contains(text.String(), "archive_bytes=") || !strings.Contains(text.String(), "nothing was written") {
		t.Fatalf("dry-run contract failed (%d -> %d):\n%s", before, after, text)
	}
	jsonOut := &bytes.Buffer{}
	if err := cmdReconcile(&clikit.Ctx{Cwd: w.Root, Stdout: jsonOut, Stderr: &bytes.Buffer{}, JSON: true}, []string{"--project", "core", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	var plan eventlog.JournalPlan
	if err := json.Unmarshal(jsonOut.Bytes(), &plan); err != nil || plan.Schema != eventlog.JournalPlanSchema || !strings.Contains(text.String(), plan.ID) {
		t.Fatalf("text/JSON identity differs: err=%v plan=%+v", err, plan)
	}
}

func TestCommandRefusesUnknownPolicyAndStalePlan(t *testing.T) {
	w := journalWorkspace(t)
	ctx := &clikit.Ctx{Cwd: w.Root, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	if err := cmdReconcile(ctx, []string{"--project", "core", "--archive-class", "old", "--dry-run"}); clikit.ExitCode(err) != 2 {
		t.Fatalf("invalid class = %v", err)
	}
	if err := cmdReconcile(ctx, []string{"--project", "core", "--apply-safe", strings.Repeat("0", 64)}); clikit.ExitCode(err) != 3 {
		t.Fatalf("unknown plan = %v", err)
	}
}

func countFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	_ = filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			count++
		}
		return nil
	})
	return count
}
