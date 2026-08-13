---
id: f-task-431-receipts-are-neither-serialized-nor-recoverable
kind: note
note_kind: finding
created: 2026-08-13T20:17:10Z
created_by: a-codex-maintainer-j6160p
about: "[[431]]"
severity: major
---
# Task 431 receipts are neither serialized nor recoverable
internal/features/queues/queues.go:147-174 and internal/features/stagegate/stagegate.go:186-223 perform an unlocked stat-before-write check; queue success mutates state before receipt persistence, while terminal paths append audit/write receipt before state mutation. Concurrent same-key calls can both mutate, and a receipt-write failure can leave mutated state with no replay checkpoint.
