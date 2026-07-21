package cli

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
)

// seedDone fabricates a completed task whose Log carries a claim→completion
// span of hours*ratio... i.e. actual = Te * ratio hours.
func seedDone(t *testing.T, dir string, n int, estimate string, ratio float64) {
	t.Helper()
	title := fmt.Sprintf("Seed history item %d", n)
	run(t, dir, 0, "task", "add", title, "--project", "p", "--estimate", estimate, "--accept", "a")
	tk := findTaskDoc(t, dir, fmt.Sprintf("%03d", n))
	tp, _ := tk.Estimate()
	start := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	end := start.Add(time.Duration(tp.Expected() * ratio * float64(time.Hour)))
	tk.Doc.Front.Set("owner", "a-root")
	tk.Doc.SetSection("Log", fmt.Sprintf("- %s claimed by a-root\n- %s completed by a-root\n",
		start.Format(time.RFC3339), end.Format(time.RFC3339)))
	if err := saveTask(tk); err != nil {
		t.Fatal(err)
	}
	w, _, err := openWorkspace(&Ctx{Cwd: dir})
	if err != nil {
		t.Fatal(err)
	}
	if err := moveTask(w, tk, model.StatusDone); err != nil {
		t.Fatal(err)
	}
}

// The P2 acceptance, verbatim from PROPOSALS: with n >= 10 in hand, the
// brief shows the calibrated range beside the PERT range, and the two
// visibly diverge where the data says they should.
func TestCalibrationGatesAtTen(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")

	// Nine completions at a consistent 1.5x overrun: still silence.
	for i := 1; i <= 9; i++ {
		seedDone(t, dir, i, "2,4,6", 1.5)
	}
	run(t, dir, 0, "task", "add", "The next piece of work", "--project", "p",
		"--estimate", "2,4,6", "--accept", "a")
	briefOut := run(t, dir, 0, "context", "010")
	if strings.Contains(briefOut, "calibrated:") {
		t.Fatalf("calibration shown at n=9 — a multiplier from anecdotes:\n%s", briefOut)
	}
	calOut := run(t, dir, 0, "calibrate")
	if !strings.Contains(calOut, "insufficient history (n=9 < 10)") {
		t.Errorf("calibrate not honest about n:\n%s", calOut)
	}

	// The tenth completion opens the gate.
	seedDone(t, dir, 11, "2,4,6", 1.5)
	briefOut = run(t, dir, 0, "context", "010")
	if !strings.Contains(briefOut, "estimate: 2/4/6 (Te 4.0)") {
		t.Fatalf("PERT line missing:\n%s", briefOut)
	}
	// Te 4, sigma 2/3: 1-sigma range 3.3–4.7 → ×1.5 → 5.0–7.0 hours.
	if !strings.Contains(briefOut, "calibrated: ~5.0–7.0h wall (×1.5 median, n=10 — time proxy, not tokens)") {
		t.Errorf("calibrated range wrong or unlabeled:\n%s", briefOut)
	}

	// The table view: banded, labeled a proxy.
	calOut = run(t, dir, 0, "calibrate")
	for _, want := range []string{"medium (≤8)", "n=10", "×1.50", "briefs now show", "time PROXY"} {
		if !strings.Contains(calOut, want) {
			t.Errorf("calibrate table missing %q:\n%s", want, calOut)
		}
	}

	// Tasks without an estimate never get a calibration line.
	run(t, dir, 0, "task", "add", "Unestimated work item", "--project", "p", "--accept", "a")
	briefOut = run(t, dir, 0, "context", "012")
	if strings.Contains(briefOut, "calibrated:") {
		t.Error("calibration line on an unestimated task")
	}
}
