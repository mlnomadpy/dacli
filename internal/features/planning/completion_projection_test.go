package planning

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
)

func TestTaskListProjectsImplementedUnlandedWithoutCountingDone(t *testing.T) {
	w, _ := taskAddEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Await landing", store.TaskOpts{Accept: []string{"verified"}})
	if err != nil {
		t.Fatal(err)
	}
	task.Doc.Front.Set("completion_state", "implemented-unlanded")
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Cwd: w.Root, Stdout: out, Stderr: &bytes.Buffer{}}
	if err := cmdTaskList(ctx, []string{"--project", "p"}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "implemented-unlanded") || !strings.Contains(got, "open") {
		t.Fatalf("text task list hid intermediate lifecycle: %q", got)
	}
	out.Reset()
	ctx.JSON = true
	if err := cmdTaskList(ctx, []string{"--project", "p"}); err != nil {
		t.Fatal(err)
	}
	var rows []taskJSON
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Status != "open" || rows[0].CompletionState != "implemented-unlanded" {
		t.Fatalf("JSON task lifecycle = %#v", rows)
	}
}
