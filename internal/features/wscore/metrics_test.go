package wscore

import (
	"testing"

	metricspkg "github.com/mlnomadpy/dacli/internal/metrics"
)

func TestWorkspaceScoreConsumesSharedMetricsReport(t *testing.T) {
	completion, retry := .8, .25
	median, total := 10.0, 20.0
	budget := 1000
	report := metricspkg.Report{
		Completion:        metricspkg.Rate{Value: &completion, Samples: 10},
		Retry:             metricspkg.Rate{Value: &retry, Samples: 8},
		Failures:          metricspkg.Failures{Classes: map[string]int{"failed": 2}, Samples: 2},
		WallTime:          metricspkg.WallTime{MedianSeconds: &median, TotalSeconds: &total, Samples: 2},
		Tokens:            metricspkg.Tokens{Budget: &budget, BudgetSamples: 1},
		HumanIntervention: metricspkg.Rate{Samples: 3},
	}
	got := scoreReport(report)
	if got.Completion.Samples != 10 || got.Retry.Samples != 8 || got.Failures.Classes["failed"] != 2 || got.WallTime.Samples != 2 || got.Tokens.BudgetSamples != 1 || got.Intervention.Samples != 3 {
		t.Fatalf("scoreReport dropped shared metrics: %+v", got)
	}
}
