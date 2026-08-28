package store

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/model"
)

// Readiness — "can this task be worked on right now?" — is asked in two
// places that must never disagree: `dacli next` (what a human or parent agent
// is told to spawn on) and the loop's BUILD phase (what a builder is actually
// handed). Both used to carry their own copy of the rule, and the loop's copy
// carried a comment asserting the two matched. They did not, on four separate
// points, and the visible symptom was `dacli next` recommending a task the
// loop silently never picked up (dacli 240). One predicate lives here, in the
// entity layer, because the feature slices may not import each other.
//
// The four disagreements and how they were settled:
//
//  1. DEP TYPES. The loop skipped SS and SF; `next` skipped only SS. Only SS
//     survives as non-blocking. dacli executes a task ATOMICALLY — one agent
//     takes it from open to done in a single run, and there is no way to start
//     a task and hold its finish — so every finish-relation (FS, FF) and the
//     start-to-finish relation collapse into the same constraint on the
//     hand-off. SS is the one relation that declares two tasks parallel-safe,
//     which is the whole reason the dep type is recorded at all.
//
//  2. UNRESOLVABLE REFS. A ref naming no task read as SATISFIED in `next` and
//     as PERMANENTLY BLOCKING in the loop. Neither side is right on its own: a
//     dangling ref is a DATA fault, not a scheduling fact. Treating it as
//     satisfied runs work whose prerequisite may not exist; treating it as
//     blocking starves a task forever on a typo. So the frontier does both
//     halves — it holds the task back (the safe side of the correctness
//     question) AND reports the ref in Problems, which `next`, the loop and
//     `dacli doctor` all surface. Blocking is only a trap while it is silent.
//
//  3. REF FORMS. `next` resolved ID, slug and seq; the loop resolved only slug
//     and %03d, so a dependency written in ID form — the form the CLI itself
//     echoes back — never matched a finished task and starved the loop
//     forever. Resolution now goes through TaskIndex, the same resolver
//     FindTask and every other command uses, including its contract that an
//     ambiguous ref is an error rather than a guess. Refs resolve within the
//     task's own project first (seq is per-project) and fall back to a
//     workspace-wide lookup for cross-project ULID refs.
//
//  4. CANDIDATE STATUS. The loop offered only `open` tasks; `next` offered
//     anything not done and not blocked, which includes ACTIVE. Open wins: an
//     active task already has an agent on it, and recommending it invites a
//     second spawn onto work in flight. A dependency, by contrast, is
//     satisfied only when DONE — both sides always agreed on that.
//
// The loop anchor and `wont` filters live here too, so that "actionable" has
// exactly one definition rather than two that happen to coincide today.

// DepProblem is a dependency reference that names no task, or names more than
// one. It is a data-integrity fault in the task file, reported rather than
// silently resolved either way — see decision 2 above.
type DepProblem struct {
	Task *Task  // the task carrying the bad ref
	Ref  string // the ref exactly as written in depends_on
	Why  string // resolver's reason: not found, or ambiguous between N tasks
}

// String renders one problem as an operator-readable line naming both the task
// and the ref, since either alone is unfixable.
func (p DepProblem) String() string {
	return fmt.Sprintf("%03d-%s depends on %q — %s", p.Task.Seq, p.Task.Slug, p.Ref, p.Why)
}

// Frontier is one evaluation of the readiness predicate over a task set.
// Ready is the workable frontier; Blocked is what is genuinely waiting on a
// dependency; Problems is what is waiting on a HUMAN to fix a ref. Callers
// that report "nothing is ready" must consult all three, or they turn a typo
// into an unexplained stall.
type Frontier struct {
	Ready    []*Task
	Blocked  []*Task
	Problems []DepProblem
}

// ProblemLines renders Problems for display. Kept here so the loop, `next` and
// `doctor` word the same fault the same way.
func (f Frontier) ProblemLines() []string {
	out := make([]string, 0, len(f.Problems))
	for _, p := range f.Problems {
		out = append(out, p.String())
	}
	return out
}

// DepBlocksStart reports whether a dependency gates handing its task to an
// agent. Everything but SS does — see decision 1 above.
func DepBlocksStart(d Dep) bool { return strings.ToUpper(d.Type) != "SS" }

// ReadyFrontier evaluates readiness over an ALL-STATUS task set (what
// ListTasks with an empty status returns): done tasks are what satisfy
// dependencies, so a caller that passes only the open ones gets an empty
// frontier and no error.
func ReadyFrontier(tasks []*Task) Frontier {
	global := NewTaskIndex(tasks)
	byProject := map[string][]*Task{}
	for _, t := range tasks {
		byProject[t.Project] = append(byProject[t.Project], t)
	}
	local := map[string]*TaskIndex{}
	for p, ts := range byProject {
		local[p] = NewTaskIndex(ts)
	}

	done := map[string]bool{}
	for _, t := range tasks {
		if t.Status == model.StatusDone {
			done[t.ID] = true
		}
	}

	var fr Frontier
	for _, t := range tasks {
		// Decision 4: only `open` work is free to hand out.
		if t.Status != model.StatusOpen {
			continue
		}
		// The standing continuous-improvement task is the review phase's
		// anchor, never implementer work.
		if t.IsLoopAnchor() {
			continue
		}
		// Aggregate tasks are derived milestones, not implementation units. Even
		// after their children finish, closure belongs to the owner-side
		// aggregate gate; handing the parent to another implementer duplicates
		// the child scope (issue #866).
		if t.IsAggregate() {
			continue
		}
		// `wont` is a recorded decision NOT to do the work — never actionable,
		// however satisfied its dependencies are (dacli 199).
		if !model.Priority(t.Priority()).Schedulable() {
			continue
		}
		blocked := false
		for _, d := range t.Deps() {
			if !DepBlocksStart(d) {
				continue
			}
			dep, err := resolveDep(local[t.Project], global, d.Ref)
			if err != nil {
				fr.Problems = append(fr.Problems, DepProblem{Task: t, Ref: d.Ref, Why: err.Error()})
				blocked = true
				continue
			}
			if !done[dep.ID] {
				blocked = true
			}
		}
		if blocked {
			fr.Blocked = append(fr.Blocked, t)
		} else {
			fr.Ready = append(fr.Ready, t)
		}
	}
	return fr
}

// resolveDep looks a dep ref up in the task's own project first, then
// workspace-wide. Seq refs (`001`, `001-slug`) are only unique per project, so
// a project-local hit must win over a same-seq task in a sibling project;
// falling back to the global index keeps cross-project ULID refs working.
func resolveDep(local, global *TaskIndex, ref string) (*Task, error) {
	if local != nil {
		if t, err := local.Find(ref); err == nil {
			return t, nil
		} else if !isNotFound(err) {
			return nil, err // ambiguous WITHIN the project — a real fault
		}
	}
	return global.Find(ref)
}

func isNotFound(err error) bool {
	var nf ErrNotFound
	return errors.As(err, &nf)
}
