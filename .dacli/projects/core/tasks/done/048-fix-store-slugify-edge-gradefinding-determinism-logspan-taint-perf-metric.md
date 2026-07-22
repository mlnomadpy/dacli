---
id: t-01KY59YVZEFEMFM1ECTTC7ETK6
kind: task
created: 2026-07-22T16:18:52Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# FIX store: Slugify edge, GradeFinding determinism, logSpan, taint perf+metric
## Acceptance
- [x] Slugify never returns empty (punctuation-only / non-ASCII titles) — no '.md' filenames or bare 'f-'/'d-' ids
- [x] GradeFinding targets the intended finding when two share a title; logSpan does not inflate the span across re-claim/re-complete cycles
- [x] Taint resolves refs via the existing TaskIndex not FindTask-per-hit (O(hits x tasks)); TaskBand does not re-scan all of RunsDir per task
- [x] taint metric over-report fixed: a workspace-scoped metric note is not marked TreeWide if WorkspaceLessons never surfaces metric notes (or make it surface them, consistently)
- [x] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T16:19:09Z claimed by a-n2q5ysnx5y
- 2026-07-22T16:40:38Z accepted by a-root
- 2026-07-22T16:40:38Z completed by a-root
- 2026-07-22T18:23:33Z status done proposed by a-n2q5ysnx5y, applied (event 01KY5AD7VFA9X2DZ7BX51PNND0)
