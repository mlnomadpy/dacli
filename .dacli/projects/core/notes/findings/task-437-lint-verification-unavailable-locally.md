---
id: f-task-437-lint-verification-unavailable-locally
kind: note
note_kind: finding
created: 2026-08-13T22:27:29Z
created_by: a-fixer-btcg7r
about: "[[437]]"
severity: minor
---
# Task 437 lint verification unavailable locally
The exact golangci-lint run could not execute because golangci-lint is not installed (zsh: command not found). gofmt -l ., go vet ./..., and GOCACHE=/private/tmp/dacli-437-go-cache go test ./... passed.
