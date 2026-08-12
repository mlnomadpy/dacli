---
id: f-claim-inference-still-under-scopes-estimate-validation-tasks-after-task-385
kind: note
note_kind: finding
created: 2026-08-12T17:02:17Z
created_by: a-codex-loop-auditor-hexawh
about: "[[392]]"
severity: major
---
# Claim inference still under-scopes estimate validation tasks after task 385
Recorded run 01KZVEDE8B5QHRGT4R6DN8RGTG spawned task 381 with Claims=internal/store even though acceptance explicitly requires spm.ThreePoint.Valid, task add/task estimate regressions, and critical-path output. Its transcript line 53 records dacli commit exit 3 refusing internal/spm, internal/features/planning, internal/features/insight, and internal/cli; the focused implementation remained uncommitted. Current internal/store/store.go:1889-1893 only infers store/execution/cli vocabulary and has no estimate/planning/spm/critical-path mapping.
