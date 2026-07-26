---
id: d-112-factored-the-anchor-predicate-into-store-task-isloopanchor-applied-in-both
kind: note
note_kind: decision
created: 2026-07-26T23:00:36Z
created_by: a-fw66q78xzb
about: [[112]]
---
# 112: factored the anchor predicate into store.Task.IsLoopAnchor, applied in both readyTasks and dacli next
## Chose
112: factored the anchor predicate into store.Task.IsLoopAnchor, applied in both readyTasks and dacli next
## Rejected
front-matter kind/marker field on the anchor task
## Because
the title-prefix marker already exists and works (orchestration.go ensureImproveTask/readyTasks); adding a store.Task.IsLoopAnchor() method plus a store.ContinuousImprovementMarker const gives both orchestration.go and insight.go a single shared predicate without a data-model migration for existing/already-filed anchor tasks that lack any kind field
