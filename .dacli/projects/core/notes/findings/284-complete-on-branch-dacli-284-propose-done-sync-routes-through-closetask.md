---
id: f-284-complete-on-branch-dacli-284-propose-done-sync-routes-through-closetask
kind: note
note_kind: finding
created: 2026-08-04T20:13:40Z
created_by: a-maintainer-rgjgzv
about: "[[284]]"
severity: major
---
# 284 complete on branch dacli/284-...: propose:done sync routes through CloseTask + verifies acceptance
Commit 5c9903e by a-maintainer-rgjgzv on branch dacli/284-route-propose-done-event-sync-closes-through-closetask-so-they-stamp-completed. Staged 3 files: internal/eventlog/sync.go, internal/eventlog/sync_test.go, internal/cli/agents_run_test.go.

ALL 5 acceptance met:
(1) sync.go apply() EventProposeStatus now branches on target==model.StatusDone: routes through store.CloseTask (writes 'completed by' stamp) instead of bare store.MoveTask (sync.go:166-207).
(2) Before the move it verifies t.Acceptance(): any unchecked box => return (false,"",nil), leaving the event PENDING — mirrors the owner path's Refusedf (planning.go:404-406). Unmet acceptance is never silently moved to done.
(3) Stamp verified via store.LogHasStamp(t,'completed by')==true in the new test, so calibration.logSpan (claim->completion) is measurable and doctor no longer sees a broken span. CloseTask attributes 'completed by' to e.Actor (the proposing agent who did the work) and guards against a mid-apply re-run appending a 2nd stamp (idempotency: LogHasStamp check before CloseTask).
(4) Regression tests: TestSyncProposeDoneRoutesThroughCloseTask (all boxes checked -> closes with stamp + claimed-by span, idempotent on re-sync) and TestSyncProposeDoneUnmetAcceptanceStaysPending (unchecked box -> NOT in done, event stays pending). Both FAIL on pre-change sync.go (verified by git-stashing the fix: 'no completed by stamp' and 'applied 1' respectively), PASS after. Also updated cli TestSpawnedChildIdentity, which had ENCODED the bug ('propose-status applies the move regardless' of an unchecked box) — it now asserts '1 applied, 1 left pending' and the task is not in done/.
(5) go build ./... , go test ./... , go vet ./... all green; gofmt -l internal/ clean.

Owner: verify + close via 'dacli accept 284' then 'dacli merge --task 284'.
