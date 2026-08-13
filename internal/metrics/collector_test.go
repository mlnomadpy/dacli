package metrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestCollectScopesUsageBudgetAndFailureToNamedWindow(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "metrics")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	writeMetricRun(t, w, "01OLD", start.Add(-time.Hour), "failed", "", "")
	writeMetricRun(t, w, "01NEW", start.Add(time.Minute), "failed", "output_tokens: 700\n", "max_tokens: 2000\n")
	until := start.Add(time.Hour)
	report := Collect(w, Window{Name: "candidate", Since: &start, Until: &until}, "")

	if report.Runs != 1 || report.Failures.Classes["failed"] != 1 {
		t.Fatalf("window leaked or dropped failure data: %+v", report)
	}
	if report.Tokens.Output == nil || *report.Tokens.Output != 700 || report.Tokens.Samples != 1 {
		t.Fatalf("usage sample missing: %+v", report.Tokens)
	}
	if report.Tokens.Budget == nil || *report.Tokens.Budget != 2000 || report.Tokens.BudgetSamples != 1 {
		t.Fatalf("budget sample missing: %+v", report.Tokens)
	}
}

func writeMetricRun(t *testing.T, w *workspace.Workspace, id string, started time.Time, outcome, usage, invocation string) {
	t.Helper()
	dir := w.RunDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{RunID: id, Started: started}); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"outcome.md": "outcome: " + outcome + "\nelapsed: 1s\n", "usage.txt": usage, "invocation.txt": invocation} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
