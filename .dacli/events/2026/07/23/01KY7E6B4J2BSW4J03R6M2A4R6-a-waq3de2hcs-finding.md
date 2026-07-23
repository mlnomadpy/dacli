---
id: 01KY7E6B4J2BSW4J03R6M2A4R6
kind: event
event_kind: finding
created: 2026-07-23T12:11:20Z
created_by: a-waq3de2hcs
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
loop BUILD phase picks tasks by seq number, ignoring MoSCoW priority and critical path

runCycle selects the cycle's batch as ready[:width] (internal/features/orchestration/orchestration.go:294-297) over a ready list that preserves ListTasks order. ListTasks sorts only by (Project, Seq) (internal/store/store.go:346-351), and readyTasks (orchestration.go:456-492) appends in that iteration order with NO priority sort. So the loop builds the lowest-Seq ready tasks every cycle: a low-seq 'could'/'should' is built before a higher-seq 'must', and the critical path is ignored. This directly undermines the loop's core promise (docstring: review->plan->implement, MoSCoW/critical-path first). The correct selection already exists and is reused nowhere here: cmdNext sorts candidates by model.Priority(..).Rank() then CPM slack then Seq (internal/features/insight/insight.go:209-218). Fix: have readyTasks (or runCycle) rank the ready frontier by priority-then-slack-then-seq before slicing to width.
