---
id: f-duplicate-task-026-existed-in-both-open-and-done-stale-open-copy-0-boxes-done
kind: note
note_kind: finding
created: 2026-07-22T15:22:35Z
created_by: a-root
severity: major
---
# Duplicate task: 026 existed in both open/ and done/ (stale open copy, 0 boxes; done copy authoritative, 3 boxes), making ref '26' ambiguous and breaking dacli ship at integrate. Root causes to harden: FindTask must not return a task twice / should prefer a single status, MoveTask must guarantee no stale source copy, and integrate/ship should resolve refs unambiguously. ship correctly fail-stopped (nothing pushed)
