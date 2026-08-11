package insight

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// Every figure must carry its DEFINITION. Two readers who disagree about what
// counts as a retry will disagree about whether the loop is improving, and
// neither will know it (issue #437 item 6).
func TestMetricsDefinesEveryFigureItPrints(t *testing.T) {
	m := metrics{
		Runs: 10, Terminal: 8, Completed: 6,
		TasksAttempted: 4, RunsPerTask: 2,
		WallMedian: 90 * time.Second, WallTotal: 12 * time.Minute,
		Tokens: 5000, TokenSamples: 3,
		TasksDone: 4, Interventions: 1,
	}
	var b bytes.Buffer
	m.render(&b)
	out := b.String()

	for _, figure := range []string{"completion rate", "retry rate", "wall time", "token cost", "intervention rate"} {
		if !strings.Contains(out, figure) {
			t.Errorf("missing figure %q:\n%s", figure, out)
		}
	}
	// Each one is followed by prose saying what it counts.
	for _, definition := range []string{
		"outcome is ok/done",      // completion
		"one attempt",             // retry
		"from spawn to outcome",   // wall
		"reported by the runtime", // tokens
		"adopted by",              // intervention
	} {
		if !strings.Contains(out, definition) {
			t.Errorf("a figure is printed without its definition (%q):\n%s", definition, out)
		}
	}
}

// A window with no measured tokens must SAY so. Printing "0 tokens" would be a
// fabricated cost, and the whole point of this command is figures a reader can
// trust.
func TestMetricsSaysWhenTokensWereNotCaptured(t *testing.T) {
	var b bytes.Buffer
	metrics{Runs: 3, Terminal: 3, Completed: 3, TokenSamples: 0}.render(&b)
	out := b.String()
	if !strings.Contains(out, "NOT CAPTURED") {
		t.Errorf("an unmeasured token cost must be named, not printed as zero:\n%s", out)
	}
	if strings.Contains(out, "0 output tokens") {
		t.Errorf("a fabricated zero cost was printed:\n%s", out)
	}
}

// Rates over an empty denominator must read n/a, not 0%. "0% completion" on a
// window with nothing terminal in it is a false alarm.
func TestMetricsReportsNotApplicableRatherThanAFalseZero(t *testing.T) {
	var b bytes.Buffer
	metrics{Runs: 2, Terminal: 0}.render(&b)
	out := b.String()
	if !strings.Contains(out, "n/a") {
		t.Errorf("an empty denominator must read n/a:\n%s", out)
	}
	if strings.Contains(out, "0% (0 of 0)") {
		t.Errorf("a rate was computed over an empty denominator:\n%s", out)
	}
	// Still-running work is reported, not silently dropped.
	if !strings.Contains(out, "still running") {
		t.Errorf("in-flight runs must be visible:\n%s", out)
	}
}
