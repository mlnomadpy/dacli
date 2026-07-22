---
id: t-01KY6104S1B31E73VV6YQCAGC8
kind: task
created: 2026-07-22T23:01:31Z
created_by: a-2jxa2ck7jh
owner: a-2jxa2ck7jh
priority: should
---
# Make dacli loop progress reflect trunk merges under --pr --auto, not accept-close
## So that
the perpetual loop's thrash guard observes real integration and cannot false-halt or run away under the default auto-merge path
## Context
Evidence: internal/features/orchestration/orchestration.go:253 computes landed = countDone()-doneBefore (StatusDone delta), documented at :206 and relied on by the Governor thrash guard (internal/features/orchestration/governor.go:56,113 'no net progress'/'0-landed cycles'). The default LAND path is 'ship --pr --auto' (:241-242); under --auto, prIntegrateTask returns landed=false and only queues GitHub auto-merge (internal/features/vcs/lifecycle.go:611-621) — nothing merges to trunk in-cycle. The StatusDone delta comes from ship's 'accept --all' closing proposed tasks (internal/features/ship/ship.go:98-106), which diverges from trunk reality: a task counts as landed while its PR is pending or fails CI. governor_test.go feeds only synthetic landed values; the real driver landed computation vs the --auto path is untested. See finding on task 084.
## Acceptance
- [ ] runCycle's progress signal fed to Governor.AfterCycle reflects branches actually merged into the integration branch (or the docstring at orchestration.go:206 and governor contract are corrected to state the metric measures acceptance-closure, not trunk landing)
- [ ] Under the default --pr --auto path, a cycle whose PRs are only queued for auto-merge (none merged in-cycle) is not counted as trunk progress by the NoProgressHalt guard
- [ ] A new orchestration test exercises the driver's runCycle/landed accounting with a fake runner whose --auto ship merges nothing, asserting the reported landed value matches trunk-merge reality
- [ ] Pure Governor decision logic is unchanged except as the fix requires; go build ./... and go test ./internal/... stay green
## Log
- 2026-07-22T23:02:04Z claimed by a-77q7eps4da
