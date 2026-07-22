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
- [ ] Slugify never returns empty (punctuation-only / non-ASCII titles) — no '.md' filenames or bare 'f-'/'d-' ids
- [ ] GradeFinding targets the intended finding when two share a title; logSpan does not inflate the span across re-claim/re-complete cycles
- [ ] Taint resolves refs via the existing TaskIndex not FindTask-per-hit (O(hits x tasks)); TaskBand does not re-scan all of RunsDir per task
- [ ] taint metric over-report fixed: a workspace-scoped metric note is not marked TreeWide if WorkspaceLessons never surfaces metric notes (or make it surface them, consistently)
- [ ] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T16:19:09Z claimed by a-n2q5ysnx5y
