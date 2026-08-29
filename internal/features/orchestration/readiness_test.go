package orchestration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

// ------------------------------------------------------------ dacli 240 ----
//
// Two predicates used to decide "is this task workable now": readyTasks here
// and insight's ready() behind `dacli next`. They disagreed on four points,
// and the loop's copy carried a comment claiming they agreed. These tests pin
// the loop's half of the now-single predicate (store.ReadyFrontier); the
// matching half is pinned in insight/readiness_test.go.

// A dependency written in ID form (`t-<ULID>`, the form `dacli task add
// --depends-on` echoes back and the form every other resolver accepts) used to
// starve the loop forever: readyTasks keyed its done-set by %03d and slug
// only, so an ID-form ref never matched a finished task and the successor was
// blocked permanently while `dacli next` happily recommended it.
func TestReadyTasksResolvesIDFormDependency(t *testing.T) {
	w := loopEnv(t)
	dep, err := store.CreateTask(w, "a-root", "p", "Prerequisite", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, dep, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	succ, err := store.CreateTask(w, "a-root", "p", "Successor", store.TaskOpts{
		Accept:    []string{"a"},
		DependsOn: []string{dep.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := readyTasks(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(ready, succ.Seq) {
		t.Fatalf("a task whose ID-form dependency is DONE must be ready — an unresolvable-looking ref starves the loop forever; got %v", seqs(ready))
	}
}

func TestProjectLoopResolvesCrossProjectDependencyWithoutSchedulingSiblingWork(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateProject(w, "a-root", "Other", "q", "goal", ""); err != nil {
		t.Fatal(err)
	}
	dep, err := store.CreateTask(w, "a-root", "q", "External prerequisite", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, dep, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	sibling, err := store.CreateTask(w, "a-root", "q", "Unrelated sibling work", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	succ, err := store.CreateTask(w, "a-root", "p", "Successor", store.TaskOpts{
		Accept: []string{"a"}, DependsOn: []string{dep.ID},
	})
	if err != nil {
		t.Fatal(err)
	}

	fr, err := readyFrontier(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(fr.Ready, succ.Seq) {
		t.Fatalf("cross-project done dependency did not satisfy successor: problems=%v", fr.ProblemLines())
	}
	for _, got := range fr.Ready {
		if got.ID == sibling.ID || got.Project != "p" {
			t.Fatalf("project loop leaked sibling-project task: %s/%s", got.Project, got.ID)
		}
	}
}

// The same ref in %03d form must keep working — the resolver widened, it did
// not move.
func TestReadyTasksStillBlocksOnUnfinishedSeqDependency(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Prerequisite", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	succ, err := store.CreateTask(w, "a-root", "p", "Successor", store.TaskOpts{
		Accept:    []string{"a"},
		DependsOn: []string{"001"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := readyTasks(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if containsSeq(ready, succ.Seq) {
		t.Fatalf("a task whose FS dependency is still open must not be ready; got %v", seqs(ready))
	}
}

// SF (start-to-finish) used to be waved through here but blocked in `next`.
// dacli runs a task atomically — one agent takes it from open to done in a
// single run — so every relation that is not SS gates the hand-off.
func TestReadyTasksBlocksOnUnfinishedSFDependency(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Prerequisite", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	succ, err := store.CreateTask(w, "a-root", "p", "Successor", store.TaskOpts{
		Accept:    []string{"a"},
		DependsOn: []string{"001:SF"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := readyTasks(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if containsSeq(ready, succ.Seq) {
		t.Fatalf("an unfinished SF dependency must block: dacli has no way to start a task and hold its finish; got %v", seqs(ready))
	}
}

// SS is the one relation that declares two tasks parallel-safe — it must not
// block, which is the whole reason the dep type is recorded.
func TestReadyTasksIgnoresSSDependency(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Prerequisite", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	succ, err := store.CreateTask(w, "a-root", "p", "Successor", store.TaskOpts{
		Accept:    []string{"a"},
		DependsOn: []string{"001:SS"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := readyTasks(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(ready, succ.Seq) {
		t.Fatalf("SS permits overlap and must never block; got %v", seqs(ready))
	}
}

// A dep ref naming nothing is a DATA fault. The loop holds the task back
// rather than running work whose prerequisite may not exist — but it must SAY
// so, because silent starvation on a typo is the failure 240 was filed for.
func TestReadyFrontierSurfacesUnresolvableDependency(t *testing.T) {
	w := loopEnv(t)
	succ, err := store.CreateTask(w, "a-root", "p", "Successor", store.TaskOpts{
		Accept:    []string{"a"},
		DependsOn: []string{"999-no-such-task"},
	})
	if err != nil {
		t.Fatal(err)
	}
	fr, err := readyFrontier(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if containsSeq(fr.Ready, succ.Seq) {
		t.Fatalf("a task whose dependency ref names nothing must not be handed to a builder; got %v", seqs(fr.Ready))
	}
	if len(fr.Problems) != 1 || fr.Problems[0].Ref != "999-no-such-task" {
		t.Fatalf("the unresolvable ref must be reported as a data problem, got %+v", fr.Problems)
	}
}

// The driver must print the data fault when its frontier is held back by one:
// "no ready work" and "no ready work because a ref is misspelled" are
// different situations and the operator can only fix the second.
func TestLoopLogsUnresolvableDependency(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Successor", store.TaskOpts{
		Accept:    []string{"a"},
		DependsOn: []string{"999-no-such-task"},
	}); err != nil {
		t.Fatal(err)
	}
	d := newDriver(w, &fakeRunner{}, &Governor{})
	if _, err := d.readyTasks(); err != nil {
		t.Fatal(err)
	}
	out := d.ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "999-no-such-task") {
		t.Fatalf("the loop must name the unresolvable dependency ref, got:\n%s", out)
	}
}

func containsSeq(ts []*store.Task, seq int) bool {
	for _, t := range ts {
		if t.Seq == seq {
			return true
		}
	}
	return false
}

func seqs(ts []*store.Task) []int {
	out := []int{}
	for _, t := range ts {
		out = append(out, t.Seq)
	}
	return out
}
