package perfbench

import (
	"fmt"
	"os"
	"testing"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// synthWS builds a throwaway workspace with nTasks tasks and nEvents events so
// the read path can be measured at more than one size — the only way to tell a
// linear walk from a quadratic one.
func synthWS(tb testing.TB, nTasks, nEvents int) *workspace.Workspace {
	tb.Helper()
	root := tb.TempDir()
	w, err := workspace.Init(root, "bench")
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-bench", "Bench", "bench", "measure", ""); err != nil {
		tb.Fatal(err)
	}
	for i := 0; i < nTasks; i++ {
		if _, err := store.CreateTask(w, "a-bench", "bench", fmt.Sprintf("Task number %d in the bench fixture", i),
			store.TaskOpts{
				Priority: "should",
				Estimate: "2,4,8",
				Context:  "Some context body that makes the file a realistic size rather than a stub.\n",
				Accept:   []string{"it works", "it is tested"},
			}); err != nil {
			tb.Fatal(err)
		}
	}
	for i := 0; i < nEvents; i++ {
		if _, err := eventlog.Append(w, "a-bench", model.EventFinding, "", "agent",
			fmt.Sprintf("finding body %d — a sentence of realistic length for the parser to chew on", i)); err != nil {
			tb.Fatal(err)
		}
	}
	return w
}

// BenchmarkScaleListTasks/N reports ns/op at three sizes. Divide by N: a flat
// per-task cost is linear, a rising one is quadratic.
func BenchmarkScaleListTasks(b *testing.B) {
	for _, n := range []int{100, 400, 1600} {
		b.Run(fmt.Sprintf("tasks=%d", n), func(b *testing.B) {
			w := synthWS(b, n, 0)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := store.ListTasks(w, "", ""); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkScaleEventlogList(b *testing.B) {
	for _, n := range []int{100, 400, 1600} {
		b.Run(fmt.Sprintf("events=%d", n), func(b *testing.B) {
			w := synthWS(b, 1, n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := eventlog.List(w, eventlog.Query{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkScaleCreateTask(b *testing.B) {
	// The durable task-sequence state makes the hot allocation path independent
	// of backlog size. Each case starts from a different canonical backlog size;
	// rising allocations/op is a regression back to scan-per-create behavior.
	for _, n := range []int{100, 400, 1600, 6400} {
		b.Run(fmt.Sprintf("existing=%d", n), func(b *testing.B) {
			w := synthWS(b, n, 0)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := store.CreateTask(w, "a-bench", "bench",
					fmt.Sprintf("Another task %d", i), store.TaskOpts{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestMain(m *testing.M) { os.Exit(m.Run()) }
