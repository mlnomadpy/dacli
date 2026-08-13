---
id: d-task-431-explicitly-owns-internal-store-lock-export
kind: note
note_kind: decision
created: 2026-08-13T20:30:05Z
created_by: a-root
about: "[[431]]"
github:
  issue: 623
  repo: mlnomadpy/dacli
---
# Task 431 explicitly owns internal store lock export
## Chose
Task 431 explicitly owns internal store lock export
## Rejected
Treat earlier slice-local scope decisions as controlling after the owner expanded the live claim
## Because
the corrective spawn claim is internal/store,internal/features/queues,internal/features/stagegate; export a narrow store.WithFileLock wrapper, replace both PID-only locks, and use mdstore.WriteBytes for receipts without editing any unclaimed path
