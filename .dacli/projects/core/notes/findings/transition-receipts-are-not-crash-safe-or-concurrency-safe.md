---
id: f-transition-receipts-are-not-crash-safe-or-concurrency-safe
kind: note
note_kind: finding
created: 2026-08-13T20:10:38Z
created_by: a-root
about: "[[431]]"
severity: critical
---
# Transition receipts are not crash-safe or concurrency-safe
Review of commit e922104 found queue/stage success mutates durable state before writing the receipt, while terminal paths write a receipt before mutating durable state. A write failure can therefore cause a replay to duplicate a success or incorrectly no-op an unperformed terminal transition. transitionSeen plus os.WriteFile also permits concurrent callers with the same key to both pass the check. Replace this with a durable claim/pending/applied protocol or another atomic compare-and-commit design, and add injected-write-failure plus concurrent-duplicate tests before pushing.
