---
id: f-all-three-task-431-criteria-are-locally-observed-but-acceptance-remains-owner
kind: note
note_kind: finding
created: 2026-08-13T20:03:22Z
created_by: a-codex-maintainer-2j651b
about: "[[431]]"
severity: major
---
# All three task 431 criteria are locally observed but acceptance remains owner-gated
internal/features/queues/queues_test.go and internal/features/stagegate/stagegate_test.go cover stable-key replay no-ops, retryable versus terminal classification with persisted dead-letter files, and attributed success/retry/terminal EventRun audits. Focused and full go test pass. Mutation proof: disabling queue receipt lookup fails with 'replayed transition moved cursor to 2, want 1'; disabling stage receipt lookup fails with 'replay was not reported as a no-op'. task check is restricted to owner a-codex-loop-auditor-hxqjcg.
