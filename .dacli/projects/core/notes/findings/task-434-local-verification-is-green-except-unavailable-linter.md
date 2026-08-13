---
id: f-task-434-local-verification-is-green-except-unavailable-linter
kind: note
note_kind: finding
created: 2026-08-13T20:03:11Z
created_by: a-codex-maintainer-8c7ncp
about: "[[434]]"
severity: moderate
---
# Task 434 local verification is green except unavailable linter
gofmt -l ., go vet ./..., and go test ./... pass. golangci-lint run cannot execute because golangci-lint is not installed (exit 127). GitHub issue #437 could not be fetched because api.github.com is unreachable.
