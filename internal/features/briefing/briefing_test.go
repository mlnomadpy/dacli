package briefing

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func newWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	t.Setenv(agentid.EnvVar, "")
	w, err := workspace.Init(t.TempDir(), "briefing-test")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func newCtx(cwd string) (*clikit.Ctx, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &errb, Cwd: cwd}, &out, &errb
}

func mustTask(t *testing.T, w *workspace.Workspace) *store.Task {
	t.Helper()
	if _, err := store.CreateProject(w, agentid.RootID, "Core", "core", "", ""); err != nil {
		t.Fatal(err)
	}
	tk, err := store.CreateTask(w, agentid.RootID, "core", "do the thing", store.TaskOpts{Accept: []string{"it works"}})
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

func TestCmdContextUsage(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdContext(ctx, nil)); code != 2 {
		t.Error("context with no task ref must be a usage error")
	}
}

func TestCmdContextRejectsUnknownFlags(t *testing.T) {
	w := newWS(t)
	tk := mustTask(t, w)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdContext(ctx, []string{tk.Slug, "--budjet", "10"})); code != 2 {
		t.Error("a typo'd flag must be a usage error, not silently dropped")
	}
}

// The brief is the product — the assembled task must actually appear in the
// rendered output.
func TestCmdContextRendersBrief(t *testing.T) {
	w := newWS(t)
	tk := mustTask(t, w)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdContext(ctx, []string{tk.Slug}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "do the thing") {
		t.Errorf("rendered brief missing the task title:\n%s", out)
	}
}

// --record freezes the brief for replay: both the rendered text and the
// invocation metadata must land under a fresh run dir, or there is nothing to
// replay later.
func TestCmdContextRecordFreezesReplay(t *testing.T) {
	w := newWS(t)
	tk := mustTask(t, w)
	ctx, out, errb := newCtx(w.Root)
	if err := cmdContext(ctx, []string{tk.Slug, "--record", "--budget", "500"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errb.String(), "recorded:") {
		t.Fatalf("must report where the brief was recorded: %q", errb)
	}
	runDir := strings.TrimSpace(strings.TrimPrefix(errb.String(), "recorded: "))

	brief, err := os.ReadFile(filepath.Join(runDir, "brief.md"))
	if err != nil {
		t.Fatalf("brief.md not written: %v", err)
	}
	if string(brief) != out.String() {
		t.Error("recorded brief.md must match exactly what was printed to stdout")
	}

	meta, err := os.ReadFile(filepath.Join(runDir, "invocation.txt"))
	if err != nil {
		t.Fatalf("invocation.txt not written: %v", err)
	}
	for _, want := range []string{"task: " + tk.ID, "actor: " + agentid.RootID, "budget: 500"} {
		if !strings.Contains(string(meta), want) {
			t.Errorf("invocation.txt missing %q:\n%s", want, meta)
		}
	}
}

func TestCmdContextJSON(t *testing.T) {
	w := newWS(t)
	tk := mustTask(t, w)
	ctx, out, _ := newCtx(w.Root)
	ctx.JSON = true
	if err := cmdContext(ctx, []string{tk.Slug}); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		TaskID   string `json:"task_id"`
		Sections []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"sections"`
		Omitted []string `json:"omitted"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("--json output is not valid JSON: %v\n%s", err, out)
	}
	if decoded.TaskID != tk.ID {
		t.Errorf("task_id = %q, want %q", decoded.TaskID, tk.ID)
	}
	if decoded.Omitted == nil {
		t.Error("omitted must be an empty array, not null, for a stable JSON schema")
	}
	if len(decoded.Sections) == 0 {
		t.Error("a real task must yield at least one section")
	}
}

func TestCmdContextJSONUsageAndRejects(t *testing.T) {
	w := newWS(t)
	tk := mustTask(t, w)

	ctx, _, _ := newCtx(w.Root)
	ctx.JSON = true
	if code := clikit.ExitCode(cmdContext(ctx, nil)); code != 2 {
		t.Error("--json context with no task ref must be a usage error")
	}

	ctx2, _, _ := newCtx(w.Root)
	ctx2.JSON = true
	if code := clikit.ExitCode(cmdContext(ctx2, []string{tk.Slug, "--record"})); code != 2 {
		t.Error("--json context does not support --record and must reject it")
	}
}

func TestCommandsAreRegistered(t *testing.T) {
	if len(Commands) != 1 || Commands[0].Path != "context" {
		t.Fatalf("unexpected Commands: %+v", Commands)
	}
	if Commands[0].Run == nil || Commands[0].Brief == "" {
		t.Error("context command is missing a Run or Brief")
	}
}
