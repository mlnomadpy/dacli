---
id: f-task-431-acceptance-is-locally-verified-but-owner-gated
kind: note
note_kind: finding
created: 2026-08-13T20:06:15Z
created_by: a-codex-maintainer-tt3db3
about: "[[431]]"
severity: major
---
# Task 431 acceptance is locally verified but owner-gated
Commit e922104 passes GOCACHE=/private/tmp/dacli-431-gocache go test ./internal/features/queues ./internal/features/stagegate; replay-guard mutations fail both new tests. Criteria 1-3 could not be checked because dacli restricts task check to owner a-codex-loop-auditor-hxqjcg.
