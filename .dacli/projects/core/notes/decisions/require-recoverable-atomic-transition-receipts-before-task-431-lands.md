---
id: d-require-recoverable-atomic-transition-receipts-before-task-431-lands
kind: note
note_kind: decision
created: 2026-08-13T20:16:12Z
created_by: a-root
about: "[[431]]"
github:
  issue: 620
  repo: mlnomadpy/dacli
---
# Require recoverable atomic transition receipts before task 431 lands
## Chose
Require recoverable atomic transition receipts before task 431 lands
## Rejected
Ship commit e922104 with stat-then-write receipts
## Because
success currently mutates state before receipt persistence, terminal paths persist receipts before state mutation, and concurrent same-key callers both pass transitionSeen; the corrective implementation must serialize each queue/project, durably write pending then applied state, reconcile interrupted pending receipts, deduplicate audit events after crashes, and prove injected-write-failure plus concurrent-same-key behavior
