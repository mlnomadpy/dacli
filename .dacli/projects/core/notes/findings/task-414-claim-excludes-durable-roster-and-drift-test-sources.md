---
id: f-task-414-claim-excludes-durable-roster-and-drift-test-sources
kind: note
note_kind: finding
created: 2026-08-13T13:56:26Z
created_by: a-fixer-xdgjvd
about: "[[414]]"
severity: major
---
# Task 414 claim excludes durable roster and drift-test sources
dacli commit refused exit 3 because task 414 claims only README.md, docs/index.md, docs/RUNTIMES.md, docs/ROSTER.md, and docs/WALKTHROUGH.md; the required durable correction also needs internal/features/catalog/catalog.go, its test, and docs/support_claims_test.go. Per claim isolation those edits were removed. The generated roster therefore still reproduces catalog.go's stale statement that rw capability is unchecked, contradicting execution.go:1582. Owner must widen the claim or split the generator/test correction before acceptance.
