---
id: f-268-complete-on-branch-dacli-268-dacli-agents-now-finalizes-gone-detached-runs
kind: note
note_kind: finding
created: 2026-08-04T20:37:03Z
created_by: a-maintainer-1eed05
about: "[[268]]"
severity: moderate
---
# 268 complete on branch dacli/268-...: dacli agents now finalizes gone detached runs
Commit 689e576 by a-maintainer-1eed05. Root cause: finalizeRun (execution.go:2124) was called ONLY from cmdWait (execution.go:2078), so a 'spawn --detach' whose child exited without anyone running 'dacli wait' kept its 'outcome: running (detached)' placeholder (execution.go:678) forever, and 'dacli agents' lists only LIVE processes (liveAgents, execution.go:1957) — so a dead silent child and a working one were indistinguishable. Fix: new sweepFinishedDetached(w) (execution.go, after finalizeRun) finalizes any run still holding the 'running (detached)' placeholder whose process is gone per runStillLive (leader AND group), and cmdAgents calls it before listing, printing each finalized run including 'no visible result'. Tests TestAgentsFinalizesGoneDetachedRuns + TestAgentsLeavesLiveDetachedRunAlone (runrecord_test.go); verified by reproduction — with the sweep stashed the first test FAILS (outcome stays 'running (detached)', agents prints 'no live agents'), restored it PASSES. go build/test/vet ./... green, gofmt -l internal/ clean.
