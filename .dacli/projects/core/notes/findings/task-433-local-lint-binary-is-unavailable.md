---
id: f-task-433-local-lint-binary-is-unavailable
kind: note
note_kind: finding
created: 2026-08-13T21:25:15Z
created_by: a-fixer-fwr9f3
about: "[[433]]"
severity: moderate
---
# Task 433 local lint binary is unavailable
The exact required command golangci-lint run exits 127 with zsh: command not found: golangci-lint. gofmt -l . and GOCACHE=/private/tmp/dacli-433-go-cache go vet ./... completed cleanly before that step; full tests are being run separately.
