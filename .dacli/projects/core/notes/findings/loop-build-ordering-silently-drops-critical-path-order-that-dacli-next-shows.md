---
id: f-loop-build-ordering-silently-drops-critical-path-order-that-dacli-next-shows
kind: note
note_kind: finding
created: 2026-08-10T13:57:45Z
created_by: a-go-auditor-5yxchj
about: "[[t-01KZNYHPH9DVJJYD6P0WAVFNB5]]"
source_event: 01KZNZAPBD761KZK6PNMT2P8JM
---
# loop BUILD ordering silently drops critical-path order that dacli next shows: criticalPathSlack includes the unsized loop anchor, cmdNext excludes it
The loop's BUILD-phase ranking and 'dacli next' are documented to agree but build the CPM scheduling set differently.
- insight.go:168 (cmdNext) EXCLUDES the anchor:  if status!=done && status!=blocked && !t.IsLoopAnchor()
- orchestration.go:1826 (criticalPathSlack, used by rankByPriority at :1796-1806) does NOT: if t.Status!=done && t.Status!=blocked  { open=append... }  (no !IsLoopAnchor()). Then :1835 est,ok:=t.Estimate(); if !ok { return nil,false }.
The doc comment at orchestration.go:1812-1813 claims 'Degrades to haveCPM=false ... same as cmdNext' — it is NOT the same.

The anchor is created UNSIZED: ensureImproveTask (orchestration.go:1651) uses TaskOpts{Priority,Context,Accept} with no Estimate, status open. sizeUnestimated (orchestration.go:1953) only sizes the wave 'batch', which excludes anchors (readiness filters them via IsLoopAnchor at store/readiness.go:126), so the anchor is never given an estimate.

FAILURE SCENARIO (every steady-state cycle): the standing continuous-improvement anchor is open+unsized and the real backlog is sized. criticalPathSlack includes the unsized anchor => t.Estimate() ok=false at :1835 => returns nil,false => haveCPM=false => rankByPriority falls back to MoSCoW+Seq (:1805). Meanwhile dacli next excludes the anchor => haveCPM=true and displays critical-path/slack ordering. The operator sees critical-path order; the loop builds a different (seq) order — the exact degradation the estimate/CPM feature exists to prevent. CHEAPEST FIX: add '&& !t.IsLoopAnchor()' to the filter at orchestration.go:1826, mirroring insight.go:168; add a regression test asserting criticalPathSlack returns haveCPM=true with an unsized anchor present but all real tasks sized.
