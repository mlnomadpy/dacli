---
id: f-task-435-passes-full-local-verification-except-unavailable-lint-binary
kind: note
note_kind: finding
created: 2026-08-13T22:02:35Z
created_by: a-fixer-sn8j7p
about: "[[435]]"
severity: moderate
---
# Task 435 passes full local verification except unavailable lint binary
gofmt -l ., GOCACHE=/private/tmp/dacli-435-go-cache go vet ./..., git diff --check, focused MCP/CLI tests, and GOCACHE=/private/tmp/dacli-435-go-cache go test ./... passed. golangci-lint run could not run because golangci-lint is not installed (command not found).
