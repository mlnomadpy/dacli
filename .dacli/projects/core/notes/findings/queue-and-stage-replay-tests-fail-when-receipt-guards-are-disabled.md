---
id: f-queue-and-stage-replay-tests-fail-when-receipt-guards-are-disabled
kind: note
note_kind: finding
created: 2026-08-13T20:02:58Z
created_by: a-codex-maintainer-tt3db3
about: "[[431]]"
severity: major
---
# Queue and stage replay tests fail when receipt guards are disabled
Mutation evidence: replacing both receipt checks with false made TestQueueTransitionReplayFailuresAndAudit fail at queues_test.go:33 with cursor 2, want 1, and TestStageTransitionReplayFailuresAndAudit fail at stagegate_test.go:29 because replay advanced again instead of reporting no-op.
