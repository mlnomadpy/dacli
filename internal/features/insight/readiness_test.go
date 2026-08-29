package insight

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

func TestNextProjectScopeUsesWorkspaceDependencyTruthWithoutLeakingSiblingWork(t *testing.T) {
	w, ctx := doctorEnv(t)
	if _, err := store.CreateProject(w, "a-root", "Q", "q", "g", ""); err != nil {
		t.Fatal(err)
	}
	dep, err := store.CreateTask(w, "a-root", "q", "External prerequisite", store.TaskOpts{Accept: []string{"done"}, Estimate: "1,2,3"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, dep, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	sibling, err := store.CreateTask(w, "a-root", "q", "Sibling implementation", store.TaskOpts{Accept: []string{"done"}, Estimate: "1,2,3"})
	if err != nil {
		t.Fatal(err)
	}
	succ, err := store.CreateTask(w, "a-root", "p", "Cross project successor", store.TaskOpts{Accept: []string{"done"}, Estimate: "1,2,3", DependsOn: []string{dep.ID}})
	if err != nil {
		t.Fatal(err)
	}
	ctx.JSON = true
	if err := cmdNext(ctx, []string{"--project", "p"}); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Schema          string `json:"schema"`
		Recommendations []struct {
			ID      string `json:"id"`
			Project string `json:"project"`
		} `json:"recommendations"`
		Problems []string `json:"problems"`
	}
	if err := json.Unmarshal(ctx.Stdout.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatalf("next JSON: %v\n%s", err, ctx.Stdout.(*bytes.Buffer))
	}
	if got.Schema != "next/v1" || len(got.Problems) != 0 || len(got.Recommendations) != 1 || got.Recommendations[0].ID != succ.ID || got.Recommendations[0].Project != "p" {
		t.Fatalf("project-scoped next = %+v, want only successor %s (not sibling %s)", got, succ.ID, sibling.ID)
	}
}

// ------------------------------------------------------------ dacli 240 ----
//
// `dacli next` and the loop's own build frontier used to run two different
// readiness predicates, with a comment in the loop asserting they matched.
// These tests pin `next`'s half of the now-single predicate
// (store.ReadyFrontier); the loop's half is pinned in
// orchestration/readiness_test.go.

// An unresolvable dep ref used to read as SATISFIED here (`ok && !done[...]`)
// and as PERMANENTLY BLOCKING in the loop — so `next` recommended a task the
// loop would never pick up. The ref names nothing, which is a data fault:
// hold the task back AND name the ref.
func TestNextRefusesTaskWithUnresolvableDependency(t *testing.T) {
	w, ctx := doctorEnv(t)
	tk, err := store.CreateTask(w, "a-root", "p", "Depends on a typo", store.TaskOpts{
		Accept:    []string{"done"},
		DependsOn: []string{"999-no-such-task"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdNext(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if strings.Contains(out, tk.Slug) {
		t.Fatalf("next must not recommend a task whose dependency ref names nothing — the loop will never pick it up; got:\n%s", out)
	}
	note := ctx.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(note, "999-no-such-task") {
		t.Fatalf("next must name the unresolvable ref rather than silently dropping the task; got stderr:\n%s", note)
	}
}

// An ACTIVE task already has an agent on it. The loop excluded it from its
// frontier; `next` recommended it, inviting a second spawn on work in flight.
func TestNextSkipsActiveTask(t *testing.T) {
	w, ctx := doctorEnv(t)
	tk, err := store.CreateTask(w, "a-root", "p", "Already being built", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tk, model.StatusActive); err != nil {
		t.Fatal(err)
	}
	if err := cmdNext(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if strings.Contains(out, tk.Slug) {
		t.Fatalf("next must not recommend a task that is already active — it is not free to spawn on; got:\n%s", out)
	}
}

// The overview's two-line taste runs the same predicate, or the front page and
// `next` contradict each other.
func TestOverviewReadyNowSkipsActiveTask(t *testing.T) {
	w, _ := doctorEnv(t)
	tk, err := store.CreateTask(w, "a-root", "p", "Already being built", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tk, model.StatusActive); err != nil {
		t.Fatal(err)
	}
	for _, line := range readyNow(w, 5) {
		if strings.Contains(line, tk.Slug) {
			t.Fatalf("overview must not list an active task as ready now; got %q", line)
		}
	}
}

// A dependency written in ID form resolves here and must resolve identically
// in the loop — this is the agreeing half of the orchestration regression.
func TestNextRecommendsTaskWhoseIDFormDependencyIsDone(t *testing.T) {
	w, ctx := doctorEnv(t)
	dep, err := store.CreateTask(w, "a-root", "p", "Prerequisite", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, dep, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	succ, err := store.CreateTask(w, "a-root", "p", "Successor", store.TaskOpts{
		Accept:    []string{"done"},
		DependsOn: []string{dep.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdNext(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if out := ctx.Stdout.(*bytes.Buffer).String(); !strings.Contains(out, succ.Slug) {
		t.Fatalf("a task whose ID-form dependency is done is ready; got:\n%s", out)
	}
}

// A dep ref naming nothing is exactly the kind of data-integrity fault doctor
// already reports (duplicate task files, broken calibration spans). Surfacing
// it there is what makes "block on an unresolvable ref" safe rather than a
// silent starvation trap.
func TestDoctorReportsUnresolvableDependencyRef(t *testing.T) {
	w, ctx := doctorEnv(t)
	tk, err := store.CreateTask(w, "a-root", "p", "Depends on a typo", store.TaskOpts{
		Accept:    []string{"done"},
		DependsOn: []string{"999-no-such-task"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "unresolvable-dependency") {
		t.Fatalf("expected an unresolvable-dependency finding, got:\n%s", out)
	}
	if !strings.Contains(out, "999-no-such-task") || !strings.Contains(out, tk.Slug) {
		t.Fatalf("the finding must name both the task and the dangling ref, got:\n%s", out)
	}
}
