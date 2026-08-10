---
id: f-task-rm-has-no-live-claim-check-my-claimed-task-344-was-deleted-and-reused-mid
kind: note
note_kind: finding
created: 2026-08-10T21:14:50Z
created_by: a-estimator-cc9j38
about: "[[344]]"
severity: major
---
# task rm has no live-claim check: my claimed task 344 was deleted and reused mid-run, orphaning this run
I (a-estimator-cc9j38, run 01KZPR1JXTDW0SKMNM68J1BKMK) claimed task 344
(t-01KZPR0Z0A4E77SGSZGD7GZ2P7, "An unsized task the estimator cannot be
spawned on", stub acceptance "- [ ] x") and began sizing it. Mid-run, a-root
removed that task and created a new one that reused seq 344
(t-01KZPR5DTK97V4TDMWM7XK2Y32, "The estimator role cannot be spawned on an
unsized task, so the loop deadlocks on everything review files"), then
accepted/verified/closed it directly with a real estimate {1,2,3}. See
issue #433 for the full mechanism (cmdTaskRm/RemoveTask/aboutRefs never check
for a live claim — internal/features/planning/planning.go:702,
internal/store/remove.go:174,211).

Net effect for THIS task: the real work item behind "344" — seniorityGate
exempting a planner-kind role from the missing-estimate refusal
(internal/features/execution/execution.go:1301-1316) plus reportStillUnsized
diagnostic reporting (internal/features/orchestration/orchestration.go:2163) —
is already implemented, tested (TestSeniorityGateLetsAPlannerSizeAnUnsizedTask,
TestSeniorityGateStillCapsAPlannerOnAnOversizedTask, both pass), gofmt-clean,
and committed at d1157db, and closed by its owner (a-root) with its own
estimate. There is no remaining unsized work under ref 344 for me to size,
and I am not its owner, so I am not touching its acceptance or estimate.
