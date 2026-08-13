---
id: f-task-431-branch-duplicates-lower-layer-durability-primitives-despite-recorded
kind: note
note_kind: finding
created: 2026-08-13T20:25:56Z
created_by: a-codex-maintainer-y0z09r
about: "[[431]]"
severity: major
---
# Task 431 branch duplicates lower-layer durability primitives despite recorded reuse constraint
internal/features/queues/transitions.go:49-127 and internal/features/stagegate/transitions.go:44-147 implement feature-local PID/signal locks; their receipt writers also use local temp+rename while internal/mdstore/mdstore.go:626 provides durable WriteBytes and internal/store/store.go:833 keeps acquireFileLock private. The brief explicitly requires reuse, but exporting the store wrapper would touch a path outside the claimed queue/stage slices.
