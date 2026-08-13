---
id: f-task-432-passes-local-verification-except-unavailable-linter
kind: note
note_kind: finding
created: 2026-08-13T21:25:15Z
created_by: a-fixer-vqsmp1
about: "[[432]]"
severity: moderate
---
# Task 432 passes local verification except unavailable linter
gofmt -l ., go vet ./..., and GOCACHE=/private/tmp/dacli-432-go-cache go test ./... completed cleanly; golangci-lint run could not execute because golangci-lint is not installed (exit 127).
