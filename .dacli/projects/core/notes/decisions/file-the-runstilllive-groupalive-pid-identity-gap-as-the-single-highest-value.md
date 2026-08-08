---
id: d-file-the-runstilllive-groupalive-pid-identity-gap-as-the-single-highest-value
kind: note
note_kind: decision
created: 2026-08-04T16:32:02Z
created_by: a-go-auditor-a2hqh6
about: "[[280]]"
---
# File the runStillLive/GroupAlive PID-identity gap as the single highest-value change
## Chose
File the runStillLive/GroupAlive PID-identity gap as the single highest-value change
## Rejected
Filing the detached-run token escape (a wave run whose usage.txt is finalized after runCycle's LatestRunID cursor advances is dropped from every future RunsTokensSince, orchestration.go:522-523 / calibration.go:326-338), or the loop ignoring 'dacli wait' errors and never reaping a >1h-hung wave (orchestration.go:594)
## Because
The token-escape lead is real but narrow and overlaps the already-fixed idle-charge class (ChargeIdleTokens exists precisely for that family), risking a near-duplicate refusal; and in the loop the common case is covered because runCycle waits synchronously before the deferred RunsTokensSince, so the escape needs a >3600s hang. The 'wait error ignored' item is design-adjacent (wait's message tells the operator to kill) and harder to scope. The runStillLive gap is the cleanest evidence-backed defect: it is the exact un-hardened SIBLING of task 049's accepted leader-identity fix (leader is identity-checked via AliveRecord; the GroupAlive fallback at execution.go:2015 is not), it is priority-1 record-disagrees-with-reality ('runs prune' prints 'still live' for a finished agent) plus priority-4 pid-reuse lifecycle, the recycle signature (alive leader PID, mismatched start time) is distinguishable from the legitimate dacli-177 dead-leader case so the fix cannot regress that test, and it is unit-testable with a fabricated Record — a one-sitting scoped change.
