package perfbench

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var createFixtureTask = store.CreateTask

// synthWSMeasured keeps fixture construction visible without charging it to
// the benchmark operation. Task fixtures are seeded in the canonical markdown
// representation, then the final task is created through the production path.
// That last create validates the files through the real sequence observation
// and leaves the durable acceleration state exactly as a normal workspace
// would. Calling CreateTask for every seed used to make the 6,400-task case
// take minutes while measuring fixture construction rather than CreateTask.
func synthWSMeasured(tb testing.TB, nTasks, nEvents int) (*workspace.Workspace, time.Duration) {
	tb.Helper()
	started := time.Now()
	root := tb.TempDir()
	w, err := workspace.Init(root, "bench")
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-bench", "Bench", "bench", "measure", ""); err != nil {
		tb.Fatal(err)
	}
	if nTasks > 1 {
		if err := os.MkdirAll(w.TasksDir("bench", model.StatusOpen), 0o755); err != nil {
			tb.Fatal(err)
		}
	}
	for seq := 1; seq < nTasks; seq++ {
		if err := writeFixtureTask(w, seq); err != nil {
			tb.Fatal(err)
		}
	}
	if nTasks > 0 {
		if _, err := createFixtureTask(w, "a-bench", "bench", fmt.Sprintf("Task number %d in the bench fixture", nTasks),
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
	return w, time.Since(started)
}

func writeFixtureTask(w *workspace.Workspace, seq int) error {
	title := fmt.Sprintf("Task number %d in the bench fixture", seq)
	d := &mdstore.Doc{}
	d.Front.Set("id", fmt.Sprintf("t-bench-%06d", seq))
	d.Front.Set("kind", string(model.KindTask))
	d.Front.Set("created", "2026-01-01T00:00:00Z")
	d.Front.Set("created_by", "a-bench")
	d.Front.Set("owner", "a-bench")
	d.Front.Set("priority", "should")
	d.Front.Set("estimate", "{optimistic: 2, probable: 4, pessimistic: 8}")
	d.Sections = []mdstore.Section{
		{Level: 1, Title: title},
		{Level: 2, Title: "Context", Content: "Some context body that makes the file a realistic size rather than a stub.\n"},
		{Level: 2, Title: "Acceptance", Content: mdstore.RenderCheckboxes([]mdstore.Checkbox{{Text: "it works"}, {Text: "it is tested"}})},
		{Level: 2, Title: "Log"},
	}
	name := fmt.Sprintf("%03d-task-number-%s-in-the-bench-fixture.md", seq, strconv.Itoa(seq))
	// Fixture files do not need the crash-durable temp+fsync+rename transaction
	// exercised by mdstore tests. Render with the production codec and validate
	// through ListTasks/CreateTask, but avoid paying thousands of durability
	// barriers before the benchmark itself can start.
	return os.WriteFile(filepath.Join(w.TasksDir("bench", model.StatusOpen), name), []byte(mdstore.Render(d)), 0o644)
}

// BenchmarkScaleListTasks/N reports ns/op at three sizes. Divide by N: a flat
// per-task cost is linear, a rising one is quadratic.
func BenchmarkScaleListTasks(b *testing.B) {
	for _, n := range []int{100, 400, 1600} {
		b.Run(fmt.Sprintf("tasks=%d", n), func(b *testing.B) {
			w, setup := synthWSMeasured(b, n, 0)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := store.ListTasks(w, "", ""); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(setup.Microseconds())/1000, "setup-ms")
		})
	}
}

func BenchmarkScaleEventlogList(b *testing.B) {
	for _, n := range []int{100, 400, 1600} {
		b.Run(fmt.Sprintf("events=%d", n), func(b *testing.B) {
			w, setup := synthWSMeasured(b, 1, n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := eventlog.List(w, eventlog.Query{}); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(setup.Microseconds())/1000, "setup-ms")
		})
	}
}

func BenchmarkScaleCreateTask(b *testing.B) {
	// The durable task-sequence state makes the hot allocation path independent
	// of backlog size. Each case starts from a different canonical backlog size;
	// rising allocations/op is a regression back to scan-per-create behavior.
	baselineAllocs := 0.0
	for _, n := range []int{100, 400, 1600, 6400} {
		b.Run(fmt.Sprintf("existing=%d", n), func(b *testing.B) {
			w, setup := synthWSMeasured(b, n, 0)
			probe := 0
			allocs := testing.AllocsPerRun(3, func() {
				probe++
				if _, err := store.CreateTask(w, "a-bench", "bench", fmt.Sprintf("Allocation probe %d", probe), store.TaskOpts{}); err != nil {
					b.Fatal(err)
				}
			})
			if baselineAllocs == 0 {
				baselineAllocs = allocs
			} else if allocs > baselineAllocs*1.10 {
				b.Fatalf("CreateTask allocations grew with backlog: baseline %.1f, existing=%d %.1f", baselineAllocs, n, allocs)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := store.CreateTask(w, "a-bench", "bench",
					fmt.Sprintf("Another task %d", i), store.TaskOpts{}); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(allocs, "create-allocs")
			b.ReportMetric(float64(setup.Microseconds())/1000, "setup-ms")
		})
	}
}

func TestSyntheticScaleFixtureUsesOneProductionCreate(t *testing.T) {
	original := createFixtureTask
	calls := 0
	createFixtureTask = func(w *workspace.Workspace, actor, project, title string, opts store.TaskOpts) (*store.Task, error) {
		calls++
		return original(w, actor, project, title, opts)
	}
	defer func() { createFixtureTask = original }()

	w, setup := synthWSMeasured(t, 100, 0)
	if calls != 1 {
		t.Fatalf("production CreateTask calls during setup = %d, want 1; repeated creates make fixture setup quadratic", calls)
	}
	tasks, err := store.ListTasks(w, "bench", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 100 || tasks[len(tasks)-1].Seq != 100 {
		t.Fatalf("seeded fixture is not canonical: count=%d last=%+v", len(tasks), tasks[len(tasks)-1])
	}
	if setup <= 0 {
		t.Fatal("fixture setup duration was not reported")
	}
}

func TestMain(m *testing.M) { os.Exit(m.Run()) }
