---
id: d-store-verification-evidence-in-a-dedicated-task-section-through-a-shared-store
kind: note
note_kind: decision
created: 2026-08-13T21:18:43Z
created_by: a-fixer-3hxnxc
about: "[[432]]"
github:
  issue: 634
  repo: mlnomadpy/dacli
---
# Store verification evidence in a dedicated task section through a shared store schema
## Chose
Store verification evidence in a dedicated task section through a shared store schema
## Rejected
Keep extending acceptance's rendered Log sentence or duplicate parsing in acceptance and planning
## Because
accept and task-check are isolated feature slices; a store-level typed record lets both enforce provenance and lets old Log-only tasks remain readable without invented fields
