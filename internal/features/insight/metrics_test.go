package insight

import (
	"bytes"
	"strings"
	"testing"

	metricspkg "github.com/mlnomadpy/dacli/internal/metrics"
)

func TestMetricsDefinesEveryFigureItPrints(t *testing.T) {
	completion, retry, intervention := .75, .25, .1
	median, total := 60.0, 240.0
	tokens := int64(5000)
	report := metricspkg.Report{
		Window: metricspkg.Window{Name: "baseline"}, Runs: 5, TerminalRuns: 4,
		Completion:        metricspkg.Rate{Value: &completion, Samples: 4},
		Retry:             metricspkg.Rate{Value: &retry, Samples: 4},
		Failures:          metricspkg.Failures{Classes: map[string]int{"failed": 1}, Samples: 1},
		WallTime:          metricspkg.WallTime{MedianSeconds: &median, TotalSeconds: &total, Samples: 4},
		Tokens:            metricspkg.Tokens{Output: func() *int { v := int(tokens); return &v }(), Samples: 3},
		HumanIntervention: metricspkg.Rate{Value: &intervention, Samples: 10},
	}
	var out bytes.Buffer
	renderMetrics(&out, report)
	for _, want := range []string{"completion rate", "retry rate", "wall time", "token budget", "failure classes", "intervention rate", "sample"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q:\n%s", want, out.String())
		}
	}
}

func TestMetricsDoesNotFabricateMissingTokens(t *testing.T) {
	var out bytes.Buffer
	renderMetrics(&out, metricspkg.Report{Window: metricspkg.Window{Name: "empty"}, Failures: metricspkg.Failures{Classes: map[string]int{}}})
	if !strings.Contains(out.String(), "NOT CAPTURED (0 samples)") || strings.Contains(out.String(), "0 output tokens") {
		t.Fatalf("missing token data was not represented honestly:\n%s", out.String())
	}
}
