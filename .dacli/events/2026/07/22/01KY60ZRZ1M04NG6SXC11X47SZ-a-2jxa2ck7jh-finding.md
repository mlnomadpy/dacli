---
id: 01KY60ZRZ1M04NG6SXC11X47SZ
kind: event
event_kind: finding
created: 2026-07-22T23:01:19Z
created_by: a-2jxa2ck7jh
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
loop progress metric counts accept-close, not trunk merges, under --pr --auto

orchestration.go:253 computes the cycle's progress as landed = countDone() - doneBefore (StatusDone delta). The docstring (orchestration.go:206) and the Governor thrash-guard contract (governor.go:56,113 'no net progress'/'0-landed cycles') describe this as tasks 'landed on trunk'. But the loop's default LAND path is ship --pr --auto (orchestration.go:241-242), and under --auto prIntegrateTask returns landed=false and only QUEUES GitHub native auto-merge (lifecycle.go:611-621) — no branch merges to trunk during the cycle. The StatusDone delta therefore comes from ship's accept --all closing proposed tasks (ship.go:98-106), which is independent of, and can diverge from, the actual trunk merge: a task counts as 'landed' while its PR is still pending CI or later fails to merge. Net effect: the NoProgressHalt thrash guard — the loop's one guard against a runaway/stalled perpetual run — is blind to real trunk-integration under the default --auto path. governor_test.go only feeds synthetic landed values; the driver's real landed computation vs the --auto path is untested.
