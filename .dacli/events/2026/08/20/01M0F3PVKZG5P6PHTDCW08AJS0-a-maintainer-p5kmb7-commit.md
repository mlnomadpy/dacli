---
id: 01M0F3PVKZG5P6PHTDCW08AJS0
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-20T08:13:26Z
created_by: a-maintainer-p5kmb7
about: "[[t-01M0AEG5K7JF96HV0RJ5K17NJN]]"
origin: agent
applied: true
checksum: sha256:168664daf124fa6275ddc57dfd9b5b32a25630413fbdccabd47780fa17e9e90e
---
85056f9 t-01M0AEG5K7JF96HV0RJ5K17NJN: add explicit read snapshots and performance budgets

Separate brief filesystem loading from pure rendering, reuse indexed task
views across orchestration phases, and replace fixed-defect archaeology with
generated scaling and relative allocation guards.

Mutation: disabling TaskSnapshot.Invalidate made
TestTaskSnapshotRefusesInvalidStateAndRefreshSeesTransition fail with a nil
TaskIndex panic at internal/store/snapshot.go:42.
role: maintainer
