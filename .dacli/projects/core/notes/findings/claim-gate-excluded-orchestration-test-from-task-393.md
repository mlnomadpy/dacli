---
id: f-claim-gate-excluded-orchestration-test-from-task-393
kind: note
note_kind: finding
created: 2026-08-12T19:06:32Z
created_by: a-codex-maintainer-cr0hke
about: "[[393]]"
severity: minor
---
# Claim gate excluded orchestration test from task 393
dacli commit returned exit 3 for internal/features/orchestration/claim_test.go outside claims [internal/store, internal/spm, internal/features/planning, internal/features/insight, internal/cli]. I did not retry or force it; the task-381 PathsOverlap regression now lives in internal/store/claimhints_test.go, alongside existing orchestration coverage proving build spawns carry store.ClaimHints.
