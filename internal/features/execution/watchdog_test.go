package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
)

func TestFinalizePreservesRecordedTimeout(t *testing.T) {
	w := newExecWS(t)
	rec := procmon.Record{RunID: runID(8), Child: "a-timeout", PID: 1 << 30, PGID: 1 << 30, Started: time.Now().Add(-time.Minute), Timeout: 10 * time.Second}
	runDir := w.RunDir(rec.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markTimedOut(w, rec); err != nil {
		t.Fatal(err)
	}
	finalizeRun(w, rec)
	raw, err := os.ReadFile(filepath.Join(runDir, "outcome.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "outcome: timed out (detached)") {
		t.Fatalf("finalization overwrote watchdog verdict:\n%s", raw)
	}
}
