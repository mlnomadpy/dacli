package dashboard

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// BenchmarkProjectSurface500Tasks is the representative large-workspace
// measurement behind issue #932. The setup is outside the timers; the three
// cases compare the slim project summary used by the Vue heartbeat, one
// explicitly selected graph, and the legacy combined compatibility snapshot.
func BenchmarkProjectSurface500Tasks(b *testing.B) {
	w, err := workspace.Init(b.TempDir(), "a-root")
	if err != nil {
		b.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
		b.Fatal(err)
	}
	for i := 1; i <= 500; i++ {
		if _, err := store.CreateTask(w, "a-root", "core", fmt.Sprintf("Task %03d", i), store.TaskOpts{
			Accept: []string{"observable outcome"}, Estimate: "1,2,3",
		}); err != nil {
			b.Fatal(err)
		}
	}

	benchmarkJSON(b, "project-summary", func() (any, error) { return buildProjects(w) })
	benchmarkJSON(b, "selected-graph", func() (any, error) { return buildGraphResponse(w, "core") })
	benchmarkJSON(b, "legacy-combined", func() (any, error) { return buildState(w) })
}

func benchmarkJSON(b *testing.B, name string, build func() (any, error)) {
	b.Run(name, func(b *testing.B) {
		value, err := build()
		if err != nil {
			b.Fatal(err)
		}
		payload, err := json.Marshal(value)
		if err != nil {
			b.Fatal(err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := build(); err != nil {
				b.Fatal(err)
			}
		}
		b.StopTimer()
		b.ReportMetric(float64(len(payload)), "payload_B")
	})
}
