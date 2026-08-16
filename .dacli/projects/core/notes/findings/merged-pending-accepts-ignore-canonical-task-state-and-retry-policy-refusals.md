---
id: f-merged-pending-accepts-ignore-canonical-task-state-and-retry-policy-refusals
kind: note
note_kind: finding
created: 2026-08-16T17:25:41Z
created_by: a-maintainer-psevtg
about: "[[t-01KZYW7M979TQNHD2VTA1Q9WAT]]"
severity: major
---
# Merged pending accepts ignore canonical task state and retry policy refusals
internal/features/orchestration/orchestration.go:1228 invokes accept --force for every merged journal entry before resolving its task; failures are appended unchanged at line 1240, so command criteria produce the same exit-3 retry each cycle.
