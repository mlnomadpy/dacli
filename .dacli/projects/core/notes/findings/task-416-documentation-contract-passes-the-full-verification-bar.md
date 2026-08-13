---
id: f-task-416-documentation-contract-passes-the-full-verification-bar
kind: note
note_kind: finding
created: 2026-08-13T13:55:37Z
created_by: a-fixer-devvfk
about: "[[416]]"
severity: major
---
# Task 416 documentation contract passes the full verification bar
Red evidence: GOCACHE=/private/tmp/dacli-416-gocache go test ./docs -run TestPublicSupportClaimsMatchShippedSurface failed on all seven missing collaboration-boundary phrases before docs/GITHUB.md changed. Green evidence: gofmt -l . produced no output; go vet ./... passed; golangci-lint v2.12.2 with isolated cache reported 0 issues; go test ./... passed. docs/support_claims_test.go:60-74 preserves the boundary contract. Task-check was correctly refused because owner a-root alone may check boxes.
