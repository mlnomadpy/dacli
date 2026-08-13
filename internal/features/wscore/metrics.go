package wscore

import metricspkg "github.com/mlnomadpy/dacli/internal/metrics"

// scoreReport is wscore's consumer seam for the shared metrics interface. The
// bootstrap slice does not collect durable records itself; callers hand it the
// same report insight renders, preventing a second definition of completion or
// retry from growing here as workspace scoring evolves.
type workspaceScore struct {
	Completion, Retry, Intervention metricspkg.Rate
	Failures                        metricspkg.Failures
	WallTime                        metricspkg.WallTime
	Tokens                          metricspkg.Tokens
}

func scoreReport(report metricspkg.Report) workspaceScore {
	return workspaceScore{
		Completion: report.Completion, Retry: report.Retry,
		Intervention: report.HumanIntervention, Failures: report.Failures,
		WallTime: report.WallTime, Tokens: report.Tokens,
	}
}
