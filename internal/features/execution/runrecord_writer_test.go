package execution

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestRunRecordCriticalRenameFailurePreservesOldArtifact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "outcome.md")
	if err := os.WriteFile(path, []byte("old complete outcome\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	record := openRunRecord(dir, nil)
	record.rename = func(string, string) error { return errors.New("injected rename failure") }
	if err := record.critical("outcome.md", "new outcome\n"); err == nil || !strings.Contains(err.Error(), "injected rename failure") {
		t.Fatalf("critical write error = %v, want injected rename failure", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "old complete outcome\n" {
		t.Fatalf("concurrent reader-visible artifact was truncated: %q", raw)
	}
}

func TestRunRecordBestEffortFailureLeavesDurableDiagnostic(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer
	record := openRunRecord(dir, &stderr)
	// A directory at the destination makes atomic rename fail while leaving the
	// run directory itself writable for the diagnostic channel.
	if err := os.Mkdir(filepath.Join(dir, "usage.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	record.bestEffort("usage.txt", "output_tokens: 1\n")
	raw, err := os.ReadFile(filepath.Join(dir, "diagnostics.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "could not record optional usage.txt") {
		t.Fatalf("diagnostic did not name lost enrichment: %s", raw)
	}
}

func TestFinalizeRunOutcomeFailureDoesNotCompleteProcessOrAppendExit(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "finalize-failure")
	if err != nil {
		t.Fatal(err)
	}
	runID := "01FINALIZEFAIL"
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(filepath.Join(runDir, "outcome.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{RunID: runID, Child: "a-finalize", Started: time.Now()}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	if _, err := finalizeRunChecked(w, rec); err == nil {
		t.Fatal("terminal outcome failure must visibly fail finalization")
	}
	got, err := procmon.ReadRecord(filepath.Join(runDir, "proc.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != "" {
		t.Fatalf("proc record completed despite missing terminal outcome: %q", got.Outcome)
	}
	events, err := eventlog.List(w, eventlog.Query{Actor: rec.Child})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("success event appended despite terminal record failure: %+v", events)
	}
}
