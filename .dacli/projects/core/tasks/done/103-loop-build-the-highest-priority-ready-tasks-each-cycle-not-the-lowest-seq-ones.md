---
id: t-01KY7E6MS6AFV25PHSNH2Z21YN
kind: task
created: 2026-07-23T12:11:30Z
created_by: a-waq3de2hcs
owner: a-root
priority: should
---
# loop: build the highest-priority ready tasks each cycle, not the lowest-seq ones
## Context
The autonomous loop's BUILD phase picks its per-cycle batch as ready[:width] (internal/features/orchestration/orchestration.go:294-297) over a ready frontier that is only ever sorted by (Project, Seq): ListTasks sorts by Seq (internal/store/store.go:346-351) and readyTasks preserves that order with no priority sort (orchestration.go:456-492). Result: a low-seq 'could'/'should' is built before a higher-seq 'must', and the critical path is ignored — contradicting the loop's own MoSCoW/critical-path-first charter. The correct ordering already exists in cmdNext (internal/features/insight/insight.go:209-218): model.Priority(..).Rank(), then CPM slack, then Seq. Grounded in finding 01K* 'loop BUILD phase picks tasks by seq number'.
## Acceptance
- [x] readyTasks (or runCycle before slicing to width) ranks the ready frontier by MoSCoW priority rank first, then critical-path slack when a CPM schedule is available, then Seq as the final tiebreak — mirroring cmdNext's selection so the two readouts agree on what to work on first
- [x] a regression test in internal/features/orchestration constructs a ready set where a low-seq 'could' and a high-seq 'must' are both ready and width=1, and asserts the 'must' is the one placed in the built batch
- [x] the loop never hands the standing 'Continuous improvement' anchor task to a builder (existing behavior preserved)
- [x] go build ./... clean and go test ./internal/... green
## Log
- 2026-07-23T13:35:10Z claimed by a-hkm1s8wvp9
- 2026-07-23T13:39:13Z adopted by a-root (owner a-waq3de2hcs orphaned)
- 2026-07-23T13:39:13Z accepted by a-root
- 2026-07-23T13:39:13Z completed by a-root
