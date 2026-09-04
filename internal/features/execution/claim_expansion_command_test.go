package execution

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
)

func TestClaimExpandCommandRecordsOwnerAuthorityBeforeRelaunch(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "Expand after refused write", store.TaskOpts{Accept: []string{"source changes"}, Claims: []string{"src"}})
	runID := "01CLAIMEXPANDCOMMAND00001"
	if err := os.MkdirAll(w.RunDir(runID), 0o755); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{RunID: runID, Task: task.ID, Child: "a-worker", Claims: []string{"src"}, Started: time.Unix(10, 0)}
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	if err := procmon.CompleteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), rec, "failed"); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Cwd: w.Root, Stdout: out, Stderr: &bytes.Buffer{}, JSON: true}
	if err := cmdClaimExpand(ctx, []string{"--task", task.ID, "--run", runID, "--add", "tests", "--reason", "review requires a focused regression"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), store.ClaimExpansionSchema) || !strings.Contains(out.String(), "review requires") {
		t.Fatalf("claim expansion JSON = %s", out.String())
	}
	fresh, err := store.FindTask(w, task.ID)
	if err != nil || strings.Join(fresh.Claims(), ",") != "src,tests" {
		t.Fatalf("expanded claims=%v err=%v", fresh.Claims(), err)
	}
}
