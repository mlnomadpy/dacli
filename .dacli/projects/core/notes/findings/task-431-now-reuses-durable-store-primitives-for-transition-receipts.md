---
id: f-task-431-now-reuses-durable-store-primitives-for-transition-receipts
kind: note
note_kind: finding
created: 2026-08-13T20:32:28Z
created_by: a-codex-maintainer-ttvrdm
about: "[[431]]"
severity: major
---
# Task 431 now reuses durable store primitives for transition receipts
internal/store/store.go exports WithFileLock over the existing token-owned host-aware lock; internal/features/queues/transitions.go and internal/features/stagegate/transitions.go now serialize transitions through it and persist JSON receipts with mdstore.WriteBytes. Focused race tests passed with GOCACHE=/private/tmp/dacli-431-gocache.
