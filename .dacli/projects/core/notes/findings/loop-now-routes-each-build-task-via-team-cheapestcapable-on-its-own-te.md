---
id: f-loop-now-routes-each-build-task-via-team-cheapestcapable-on-its-own-te
kind: note
note_kind: finding
created: 2026-08-04T00:11:02Z
created_by: a-fixer-sphd68
about: "[[233]]"
severity: minor
---
# loop now routes each build task via team.CheapestCapable on its own Te
orchestration.go runCycle() used to resolve ONE role per cycle (d.buildRole(), the phase-appropriate fallback) and spawn every batched task with it verbatim (orchestration.go:534,546 pre-fix). Fixed: fallbackRole/fallbackKind computed once per cycle as before, but each task in the batch now calls team.CheapestCapable(roles, fallbackKind, tp.Expected()) using ITS OWN estimate (store.Task.Estimate()); an unsized task or a roster with no other role of that kind still falls back to fallbackRole exactly as before, so untemplated/unsized-task behavior is unchanged. New test TestLoopRoutesEachTaskToCheapestCapableRoleByTe (driver_test.go) proves a small (Te 1) and large (Te 10) task in the same width-2 batch route to different roles (junior-fixer vs fixer) when the roster has a capped cheap role and an uncapped expensive one. go build ./... clean; go test ./internal/... green except the pre-existing, unrelated DACLI_AGENT test-isolation gap in internal/features/catalog (this session runs as a dacli agent; passes under -exec 'env -u DACLI_AGENT').
