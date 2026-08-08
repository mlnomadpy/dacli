---
id: f-task-136-writer-routing-already-done-by-tasks-125-127-only-comma-quote-test-was
kind: note
note_kind: finding
created: 2026-07-26T21:15:59Z
created_by: a-wxkxsvvt3y
about: [[136]]
severity: minor
---
# Task 136 writer routing already done by tasks 125/127; only comma+quote test was missing
All 5 writer call sites (store.go:297 depends_on, shortcutfiles.go:39,42 params/roles, roles.go:38-41 setList for skills/scope/out_of_scope/shortcuts/escalate_to) already route through Front.SetList, fixed by task 125 (commit a2e6cb7). Front.SetList quote-escaping was fixed by task 127 (commit 37c691b). grep for raw bracket-Join pattern in internal/store is clean. The only unmet acceptance item was a round-trip test proving an element with both a top-level comma and an embedded quote survives byte-for-byte; added TestRoleScopeRoundTripsCommaAndQuoteContainingElement in internal/store/inlinelist_test.go. go build ./... clean, go test ./internal/store/... ./internal/mdstore/... and full ./internal/... green.
