---
id: f-341-complete-on-branch-dacli-341-gaterolewip-fails-open-on-an-unreadable-agents
kind: note
note_kind: finding
created: 2026-08-10T19:57:07Z
created_by: a-fixer-2mp87m
about: "[[341]]"
severity: major
---
# 341 complete on branch dacli/341-gaterolewip-fails-open-on-an-unreadable-agents-dir-activeinrole-swallows
Commit d80cd4b (a-fixer-2mp87m). All 4 acceptance criteria met.

Root cause: ActiveInRole (internal/store/roles.go:210, pre-fix) called
`agents, _ := ListAgents(w)` — ListAgents already returns a real error when
os.ReadDir(w.AgentsDir()) fails (e.g. an unreadable/non-directory agents dir),
but ActiveInRole discarded it and returned 0, i.e. "the role has nobody in
it". gateRoleWIP (execution.go:521) then compared that 0 against the WIP cap
and passed every spawn through — a fail-OPEN gate on an I/O fault, the exact
337 class ("a gate must never certify what it could not read") that gateClaimOverlap
was already fixed to avoid for the runs dir.

Fix, three parts:

1. internal/store/roles.go: ActiveInRole now returns (int, error); it
   propagates ListAgents' error instead of swallowing it. liveChildren/
   hasFinishedRun (the runs-dir reads inside the same function) are
   deliberately UNCHANGED — they already fail SAFE (an unreadable runs dir
   makes holdsWIPSlot over-count as "still holds a slot", not under-count),
   so touching them was out of this task's scope per the brief.

2. internal/features/execution/execution.go: gateRoleWIP checks the new
   error and fails closed — `return fmt.Errorf("cannot check role %s WIP: %w", ...)`
   — mirroring gateClaimOverlap's existing precedent (execution.go:612-620,
   task 337) verbatim in shape and wording style.

3. The three other ActiveInRole callers that only report or make a
   best-effort admission check (insight.go:1153 the wip-exceeded audit,
   teamops.go:62 cmdAgentSpawn's own WIP check, teamops.go:547 `dacli team`'s
   roster display) were updated to `_, _ := store.ActiveInRole(...)`-style
   error-discarding calls so they compile — acceptance criteria explicitly
   scoped these to "still compile and behave unchanged on a readable agents
   dir," not to also fail closed. dashboard/roster.go does not call
   ActiveInRole directly (it has its own ListAgents-based census in
   activeByRole for performance, per its comment) so it needed no change.

Red-green verified by hand: reverted just gateRoleWIP's error branch (kept
`active, _ := store.ActiveInRole(...)`, dropped the `if err != nil { return ... }`),
reran TestGateRoleWIPFailsClosedOnUnreadableAgentsDir — failed exactly as
predicted ("gateRoleWIP passed on an unreadable agents dir — cannot rule out
the role already being at its WIP cap"). Restored the fix, green again.

New tests:
- internal/store/wipslot_test.go: TestActiveInRoleFailsOnUnreadableAgentsDir
  — replaces the agents dir with a regular file (ENOTDIR, the 337 technique)
  and asserts ActiveInRole returns a non-nil error.
- internal/features/execution/runrecord_test.go:
  TestGateRoleWIPFailsClosedOnUnreadableAgentsDir — same fixture, drives
  gateRoleWIP directly on a launchPlan{HasRole:true, RoleName:"junior",
  Role:team.Role{WIP:1}} and asserts a non-nil, non-zero-exit error.
- Updated every existing ActiveInRole call site (wipslot_test.go,
  teamops_test.go, roster_test.go) to the new (int, error) signature; no
  behavior change on a readable agents dir (all pre-existing assertions
  still hold with err==nil).

PROOF: go build ./... clean, go vet ./... clean, gofmt -l . clean (checked
every touched file explicitly). go test ./... : only pre-existing failures
are in internal/features/briefing (TestCatchup*) and
orchestration/noremote_test.go's TestIntoRefusesAnUnknownBranchUpFront, both
hitting this dogfood session's ambient DACLI_AGENT — the same unrelated
artifact prior cycles' fixers have documented, not touched by this change.
Every package this diff touches (store, execution, teamops, dashboard,
insight) is green.

golangci-lint could NOT be run: the binary requires interactive approval in
this sandbox, unavailable to a headless agent — flagging this gap honestly
rather than claiming a check I did not run (same limitation prior fixers in
this project have hit and documented).

Owner: dacli accept 341 (task check is gated to a-root; I could not check the
boxes myself). PR-first is off — branch
dacli/341-gaterolewip-fails-open-on-an-unreadable-agents-dir-activeinrole-swallows
is ready for accept + integrate/merge --task 341.
