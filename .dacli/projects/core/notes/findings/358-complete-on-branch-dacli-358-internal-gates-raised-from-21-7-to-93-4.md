---
id: f-358-complete-on-branch-dacli-358-internal-gates-raised-from-21-7-to-93-4
kind: note
note_kind: finding
created: 2026-08-11T11:13:31Z
created_by: a-fixer-tnnxsm
about: "[[358]]"
severity: major
---
# 358 complete on branch dacli/358: internal/gates raised from 21.7% to 93.4% coverage
Commit 469aa48 (a-fixer-tnnxsm). All 3 acceptance criteria met.

Added three new test files, no product code changed except a temporary
mutation-and-revert used to verify the fail-closed guard (see below):

- internal/gates/predicates_test.go: both outcomes (satisfied + refused) for
  every predicate kind evaluate() supports: project_sections (missing /
  present-but-empty / too-short / placeholder / filled, single and
  multi-section), glossary (min_terms floor above/below), decisions (min N,
  including that CreateNote itself refuses a rejection-free decision so the
  gate's "with a rejection" promise is exercised at the boundary), tasks
  all_have_acceptance / all_have_estimate / musts_done, risks
  rank1_have_action (rank-1 unactioned vs actioned vs rank-3 needing none),
  retro (absent / unrelated ref note / real Retro: heading), and an unknown
  predicate kind / unknown sub-argument (both must refuse, never pass).

- internal/gates/template_test.go: Load (every embedded template present,
  parsed stage/cone/phase/allow/predicates correctly, solo's zero-stage
  case), workspace-vendored-overrides-embedded (nearest-wins, no
  double-listing), Get's ErrNotFound on an unknown name, and Vendor
  (copies embedded bytes, refuses a second vendor of the same name, refuses
  vendoring an unknown embedded name).

- internal/gates/lifecycle_test.go: Attach (sets first stage/cone/phase,
  solo attaches already-Complete, unknown template name errors), Status
  (untemplated project reads Complete, evaluates the current stage's
  checks, errors when the recorded stage no longer exists in the manifest —
  "the manifest changed under it"), Advance (refuses with unmet checks
  while unsatisfied, moves stage and persists template_stage/phase once
  every check passes, no-ops on an already-complete project, and — driving
  a full template via a generic satisfyCurrentStage helper — completes and
  clears the phase gate past the last stage), Stage.AllowsKind and
  Phase.AllowsKind (empty-allow permissive, listed/unlisted kind,
  ungated-phase permissive regardless of Allows), PhaseFor on an
  unattached project, and unfilled() directly across empty / whitespace /
  TBD / TODO / too-short / genuinely-filled.

(2) Fail-closed on a read error: gates_read_error_test.go already covered
the three "tasks" quantifier gates (dacli 187's fixed vacuous-truth class).
Auditing every other quantifier-shaped predicate found the SAME class was
untested (though already fixed) for "risks: rank1_have_action"
(gates.go:456-471) — it is a universal ("every rank-1 risk has an action"),
so an I/O fault on the risks directory reading as an empty set would pass
the gate on zero risks examined. Added
TestRisksRank1PredicateFailsClosedOnUnreadableSet (same unreadable-directory
technique as the existing tasks test: replace the risks dir with a regular
file, non-ENOENT ReadDir error). Verified this is a REAL regression test,
not a vacuous one: temporarily reverted the guard (`risks, rerr := ...; if
rerr != nil { return ... }` -> bare `risks, _ := ...`), reran — failed
exactly as predicted ("risks gate passed on an unreadable risk set"),
restored the fix, confirmed green and `git diff` on gates.go clean before
committing.

Audited the other quantifier-shaped predicates (decisions, retro, glossary)
for the same defect class: they are EXISTENTIAL ("at least N have
property"/"any retro exists"), so their existing swallowed-error reads
(`notes, _ := store.ListNotes(...)`, `store.GlossaryRead` returning "" on
error) degrade an I/O fault to an empty set, which still FAILS the gate
(0 >= min N is false) rather than vacuously passing it — not exploitable
the same way, so left unchanged as out of this task's scope (no behavior
to fix, only the risks universal-quantifier gate was undertested-and-latent
in the vacuous-true sense).

(3) Coverage: internal/gates was at 21.7% (measured baseline, confirmed via
`go test ./internal/gates/... -cover` before any change). After this diff:
93.4%. Per-function breakdown (go tool cover -func) shows every previously
0%-covered function now covered: Load 91.7%, Get 85.7%, Vendor 88.9%, parse
92.3%, Attach 93.8%, Status 100%, status 90.9%, Advance 94.4%, writePhase
87.5%, PhaseFor 71.4%, both AllowsKind methods 100%, evaluate 97.1%,
unfilled 90.9%. Note: task 357 (raise the coverage floor to reality + add
per-package floors), which this task's 3rd acceptance criterion references,
is still open/unowned by this task — I did not implement the per-package
floor mechanism itself (out of this task's scope, task 357's job); 93.4%
clears any floor that mechanism could plausibly set relative to the 21.7%
baseline it was filed against.

PROOF: gofmt -l . clean, go vet ./... clean. go test ./internal/gates/...
-v: all 33 tests pass (4 pre-existing + 29 new). go test ./internal/gates/...
-cover: 93.4%. Full go test ./...: internal/gates fully green; the only
failures anywhere are the pre-existing, already-documented ambient
DACLI_AGENT dogfood-session artifacts in internal/features/briefing
(TestCatchup*), internal/features/orchestration
(TestIntoRefusesAnUnknownBranchUpFront), and internal/features/teamops
(TestAgentSpawnFailsClosedWhenTheWIPCountCannotBeRead) — none touch
internal/gates.

golangci-lint could NOT be run: the binary requires interactive approval in
this sandbox, unavailable to a headless agent — flagging this gap honestly
rather than claiming a check I did not run (same limitation prior fixers in
this project have documented).

Owner: dacli accept 358 (task check is gated to a-root; I could not check
the boxes myself). PR-first is off — branch
dacli/358-cover-internal-gates-the-stage-gate-safety-surface-sitting-at-21-7-percent
is ready for accept + integrate/merge --task 358.
