---
id: f-task-431-recoverable-receipt-verification-passes
kind: note
note_kind: finding
created: 2026-08-13T20:24:00Z
created_by: a-codex-maintainer-j6160p
about: "[[431]]"
severity: major
---
# Task 431 recoverable receipt verification passes
New tests in internal/features/queues/queues_test.go and internal/features/stagegate/stagegate_test.go inject failure while promoting pending receipts and run concurrent same-key callers. Mutation check removing queue serialization failed consistently with 'audit events = 3, want one per stable key'. Restored implementation passes focused -race, gofmt -l ., go vet ./..., and go test ./... with GOCACHE=/private/tmp/dacli-431-gocache; golangci-lint is unavailable in PATH.
