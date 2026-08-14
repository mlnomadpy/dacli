---
id: f-task-445-lint-verification-unavailable-locally
kind: note
note_kind: finding
created: 2026-08-14T00:41:56Z
created_by: a-maintainer-1w0gkw
about: "[[445]]"
severity: minor
---
# Task 445 lint verification unavailable locally
go build ./..., go test ./..., go vet ./..., and gofmt -l . passed with GOCACHE=/tmp/dacli-445-gocache. golangci-lint run could not execute because golangci-lint is not installed (command not found).
