---
id: f-lint-and-complete-repository-test-suite-were-unavailable-in-the-sandbox
kind: note
note_kind: finding
created: 2026-08-26T13:21:03Z
created_by: a-fixer-s3nggb
about: "[[t-01M0CZANK00P2B5XY6TVJNAWCK]]"
severity: moderate
---
# Lint and complete repository test suite were unavailable in the sandbox
golangci-lint is not installed. GOCACHE=/private/tmp/dacli-go-cache GOMODCACHE=/private/tmp/dacli-go-mod-cache go test ./... returned after the first six packages without reporting the remaining packages; focused and full internal/features/execution tests passed.
