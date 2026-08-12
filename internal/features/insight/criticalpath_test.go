// `critical-path` refuses rather than degrading, which is right: a CPM built
// on fabricated durations is worse than no CPM, because it looks authoritative.
// But a refusal is only useful if acting on it is one step, and this one made
// it N — it named the FIRST unestimated task and returned, so an operator with
// five unsized tasks fixed one, re-ran, hit the next, and round-tripped five
// times to learn something the first run could have said in full.
//
// It also named the fault without naming the remedy, which doctor's own
// findings all do (`dacli accept --force` is spelled out there).
package insight

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
)

func cpOut(t *testing.T, args ...string) (string, error) {
	t.Helper()
	_, ctx := doctorEnv(t)
	err := cmdCriticalPath(ctx, args)
	return ctx.Stdout.(*bytes.Buffer).String(), err
}

// TestCriticalPathNamesEveryUnestimatedTask is the regression: one run, one
// list, one round trip.
func TestCriticalPathNamesEveryUnestimatedTask(t *testing.T) {
	w, ctx := doctorEnv(t)
	titles := []string{"Alpha work unsized", "Beta work unsized", "Gamma work unsized"}
	var slugs []string
	for _, title := range titles {
		tk, err := store.CreateTask(w, "a-root", "p", title, store.TaskOpts{Accept: []string{"a"}})
		if err != nil {
			t.Fatal(err)
		}
		slugs = append(slugs, tk.Slug)
	}
	// One sized task, so the refusal is provably about the unsized ones rather
	// than about an empty or wholly unestimated workspace.
	if _, err := store.CreateTask(w, "a-root", "p", "Delta work that is sized",
		store.TaskOpts{Accept: []string{"a"}, Estimate: "1,2,3"}); err != nil {
		t.Fatal(err)
	}

	err := cmdCriticalPath(ctx, nil)
	if err == nil {
		t.Fatal("critical-path must refuse while any open task has no estimate")
	}
	for _, s := range slugs {
		if !strings.Contains(err.Error(), s) {
			t.Fatalf("every unestimated task must be named in ONE refusal; %s is missing from: %v", s, err)
		}
	}
	// The command named must be the one that SETS an estimate. `dacli estimate`
	// only reads one back and tells you to add it elsewhere, so pointing there
	// would send the reader on the same extra round trip this fix removed.
	if !strings.Contains(err.Error(), "task estimate") || !strings.Contains(err.Error(), "--estimate") {
		t.Fatalf("the refusal must name the command that actually sets an estimate, got: %v", err)
	}
}

// TestCriticalPathSchedulesOnceEverythingIsSized is the other half — the
// refusal must be about missing estimates and nothing else, or the test above
// would pass against a command that never schedules at all.
func TestCriticalPathSchedulesOnceEverythingIsSized(t *testing.T) {
	w, ctx := doctorEnv(t)
	first, err := store.CreateTask(w, "a-root", "p", "Foundation work",
		store.TaskOpts{Accept: []string{"a"}, Estimate: "2,3,4"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateTask(w, "a-root", "p", "Dependent work",
		store.TaskOpts{Accept: []string{"a"}, Estimate: "1,2,9", DependsOn: []string{first.Slug}})
	if err != nil {
		t.Fatal(err)
	}

	if err := cmdCriticalPath(ctx, nil); err != nil {
		t.Fatalf("a fully sized backlog must schedule: %v", err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "project duration") {
		t.Fatalf("expected a schedule, got:\n%s", out)
	}
	if strings.Contains(out, "Inf") || strings.Contains(out, "NaN") {
		t.Fatalf("finite accepted estimates produced a non-finite schedule:\n%s", out)
	}
	for _, s := range []string{first.Slug, second.Slug} {
		if !strings.Contains(out, s) {
			t.Fatalf("every scheduled task must appear, %s missing from:\n%s", s, out)
		}
	}
	// A chain of two has no slack anywhere — both are on the critical path. If
	// the star never appears, the command printed a table and answered nothing.
	if !strings.Contains(out, "★") {
		t.Fatalf("a two-task dependency chain is entirely critical; no star in:\n%s", out)
	}
}

// TestCriticalPathReportsAnEmptyBacklogPlainly: no open work is not an error,
// and must not read as one.
func TestCriticalPathReportsAnEmptyBacklogPlainly(t *testing.T) {
	out, err := cpOut(t)
	if err != nil {
		t.Fatalf("an empty backlog is not a fault: %v", err)
	}
	if !strings.Contains(out, "nothing open") {
		t.Fatalf("expected a plain report, got:\n%s", out)
	}
}

// TestCriticalPathIgnoresDoneWork: done tasks are not scheduled, so an
// unestimated DONE task must not block the whole command. Historical tasks
// closed before estimates existed would otherwise make critical-path
// permanently unusable in any long-lived workspace — including this one.
func TestCriticalPathIgnoresDoneWork(t *testing.T) {
	w, ctx := doctorEnv(t)
	old, err := store.CreateTask(w, "a-root", "p", "Ancient work with no estimate", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, old, "done"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTask(w, "a-root", "p", "Current sized work",
		store.TaskOpts{Accept: []string{"a"}, Estimate: "1,2,3"}); err != nil {
		t.Fatal(err)
	}

	if err := cmdCriticalPath(ctx, nil); err != nil {
		t.Fatalf("an unestimated DONE task must not block scheduling: %v", err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if strings.Contains(out, old.Slug) {
		t.Fatalf("done work must not be scheduled, got:\n%s", out)
	}
}
