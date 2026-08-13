---
id: d-keep-transition-receipts-and-dead-letters-inside-each-queue-project-feature
kind: note
note_kind: decision
created: 2026-08-13T20:02:58Z
created_by: a-codex-maintainer-tt3db3
about: "[[431]]"
github:
  issue: 619
  repo: mlnomadpy/dacli
---
# Keep transition receipts and dead letters inside each queue/project feature directory
## Chose
Keep transition receipts and dead letters inside each queue/project feature directory
## Rejected
Add shared store or eventlog schema APIs
## Because
the task claim is slice-isolated; hashed per-key receipt files make replay checks durable while the existing append-only event log supplies attributed audit records
