// Package perfbench holds read-path benchmarks measured against a real
// workspace. It contains no production code: it exists so `go test -bench .
// -benchmem ./internal/perfbench` gives a repeatable number for the functions
// nearly every dacli command pays for.
//
// By default the benchmarks run against the repo's OWN .dacli workspace (the
// scale model: ~1 project, ~240 tasks, ~380 notes, ~340 events, ~450 runs).
// Set DACLI_BENCH_WS=/path/to/repo to point them elsewhere.
package perfbench

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// realWS opens the workspace under test, skipping when none is reachable.
func realWS(tb testing.TB) *workspace.Workspace {
	tb.Helper()
	start := os.Getenv("DACLI_BENCH_WS")
	if start == "" {
		// internal/perfbench -> repo root
		wd, err := os.Getwd()
		if err != nil {
			tb.Skip(err)
		}
		start = filepath.Join(wd, "..", "..")
	}
	w, err := workspace.Find(start)
	if err != nil {
		tb.Skipf("no workspace at %s: %v", start, err)
	}
	return w
}

// firstProject is the project the per-project benchmarks read.
func firstProject(tb testing.TB, w *workspace.Workspace) string {
	ps, err := store.ListProjects(w)
	if err != nil || len(ps) == 0 {
		tb.Skip("no projects")
	}
	return ps[0].Slug
}

func BenchmarkListTasksAll(b *testing.B) {
	w := realWS(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ts, err := store.ListTasks(w, "", "")
		if err != nil {
			b.Fatal(err)
		}
		if len(ts) == 0 {
			b.Skip("empty workspace")
		}
	}
}

func BenchmarkListTasksOpen(b *testing.B) {
	w := realWS(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := store.ListTasks(w, "", model.StatusOpen); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFindTask(b *testing.B) {
	w := realWS(b)
	all, err := store.ListTasks(w, "", "")
	if err != nil || len(all) == 0 {
		b.Skip("no tasks")
	}
	ref := fmt.Sprintf("%d", all[len(all)-1].Seq)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.FindTask(w, ref); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFindTaskLoop10 is the shape a command that resolves several refs
// pays when it calls FindTask per ref instead of building a TaskIndex once.
func BenchmarkFindTaskLoop10(b *testing.B) {
	w := realWS(b)
	all, err := store.ListTasks(w, "", "")
	if err != nil || len(all) < 10 {
		b.Skip("not enough tasks")
	}
	refs := make([]string, 0, 10)
	for _, t := range all[:10] {
		refs = append(refs, fmt.Sprintf("%d", t.Seq))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, r := range refs {
			if _, err := store.FindTask(w, r); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkTaskIndexLoop10(b *testing.B) {
	w := realWS(b)
	all, err := store.ListTasks(w, "", "")
	if err != nil || len(all) < 10 {
		b.Skip("not enough tasks")
	}
	refs := make([]string, 0, 10)
	for _, t := range all[:10] {
		refs = append(refs, fmt.Sprintf("%d", t.Seq))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx, err := store.BuildTaskIndex(w)
		if err != nil {
			b.Fatal(err)
		}
		for _, r := range refs {
			if _, err := idx.Find(r); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkEventlogListAll(b *testing.B) {
	w := realWS(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := eventlog.List(w, eventlog.Query{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEventlogListPendingFindings(b *testing.B) {
	w := realWS(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := eventlog.List(w, eventlog.Query{
			Kinds: []model.EventKind{model.EventFinding}, Pending: true}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventlogListAboutLimit5 is brief.Assemble's "recent activity" query.
// Limit only short-circuits once 5 MATCHING events are collected, so a task
// with fewer than 5 events pays a full walk + parse of the whole log.
func BenchmarkEventlogListAboutLimit5(b *testing.B) {
	w := realWS(b)
	all, err := store.ListTasks(w, "", "")
	if err != nil || len(all) == 0 {
		b.Skip("no tasks")
	}
	id := all[len(all)-1].ID
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eventlog.List(w, eventlog.Query{About: id, Limit: 5}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadRoles(b *testing.B) {
	w := realWS(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := store.LoadRoles(w); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListNotesFindings(b *testing.B) {
	w := realWS(b)
	p := firstProject(b, w)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.ListNotes(w, p, model.NoteFinding); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWorkspaceLessons(b *testing.B) {
	w := realWS(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = store.WorkspaceLessons(w, "")
	}
}

func BenchmarkLoadCalibration(b *testing.B) {
	w := realWS(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = store.LoadCalibration(w)
	}
}

func BenchmarkBriefAssemble(b *testing.B) {
	w := realWS(b)
	all, err := store.ListTasks(w, "", "")
	if err != nil || len(all) == 0 {
		b.Skip("no tasks")
	}
	ref := fmt.Sprintf("%d", all[len(all)-1].Seq)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br, err := brief.Assemble(w, ref, brief.Options{})
		if err != nil {
			b.Fatal(err)
		}
		_ = br.Render()
	}
}

// BenchmarkGhmirrorListNotesPerTask reproduces ghmirror.go:650 —
// store.ListNotes called once per task inside the push loop — at the real task
// count, against the hoisted-once shape below. This is the O(tasks × notes)
// claim, measured.
func BenchmarkGhmirrorListNotesPerTask(b *testing.B) {
	w := realWS(b)
	p := firstProject(b, w)
	tasks, err := store.ListTasks(w, p, "")
	if err != nil || len(tasks) == 0 {
		b.Skip("no tasks")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range tasks {
			if _, err := store.ListNotes(w, p, model.NoteFinding); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkGhmirrorListNotesHoisted(b *testing.B) {
	w := realWS(b)
	p := firstProject(b, w)
	tasks, err := store.ListTasks(w, p, "")
	if err != nil || len(tasks) == 0 {
		b.Skip("no tasks")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		notes, err := store.ListNotes(w, p, model.NoteFinding)
		if err != nil {
			b.Fatal(err)
		}
		for range tasks {
			_ = notes
		}
	}
}

// BenchmarkNextReadPath is everything `dacli next` reads before it prints:
// one task walk, the workspace lessons, the roles. The lesson matcher then
// scores lessons against at most --parallel (default 3) candidates, so it is
// O(3 × lessons), not O(tasks × lessons).
func BenchmarkNextReadPath(b *testing.B) {
	w := realWS(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := store.ListTasks(w, "", ""); err != nil {
			b.Fatal(err)
		}
		_ = store.WorkspaceLessons(w, "")
		if _, err := store.LoadRoles(w); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCalibrationTwoWalks is the shape execution.printAdvisory +
// bandTokenBudget pay: CalibrationSamples called twice in one flow
// (execution.go:575 and execution.go:645), each a full LoadCalibration.
func BenchmarkCalibrationTwoWalks(b *testing.B) {
	w := realWS(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = store.CalibrationSamples(w)
		_ = store.CalibrationSamples(w)
	}
}

// BenchmarkMdstoreReadFile isolates the per-file parse cost every list walk
// multiplies by its file count.
func BenchmarkMdstoreReadFile(b *testing.B) {
	w := realWS(b)
	all, err := store.ListTasks(w, "", "")
	if err != nil || len(all) == 0 {
		b.Skip("no tasks")
	}
	path := all[len(all)-1].Path
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := mdstore.ReadFile(path); err != nil {
			b.Fatal(err)
		}
	}
}
