---
id: f-task-406-needs-runtime-execution-seam-completion
kind: note
note_kind: finding
created: 2026-08-13T10:50:23Z
created_by: a-root
about: "[[406]]"
severity: major
---
# Task 406 needs runtime execution seam completion
PR #561 currently contains only internal/providerpolicy, internal/store, and internal/team policy primitives. Continue the existing task branch; wire nonzero runtime exits in internal/features/execution into typed provider outcomes, enforce persisted cooldown/fallback selection before spawn, and ensure supervise/loop records and prints the same transition. Add end-to-end tests proving permanent input and policy failures never fall back. Preserve explicit opt-in role fallback and grant/capability floors.
