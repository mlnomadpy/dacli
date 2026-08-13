---
id: f-task-400-lint-gate-unavailable-in-sandbox
kind: note
note_kind: finding
created: 2026-08-12T20:22:07Z
created_by: a-codex-maintainer-w6vv23
about: "[[400]]"
severity: minor
---
# Task 400 lint gate unavailable in sandbox
Required golangci-lint run could not execute because golangci-lint is not installed (zsh exit 127). gofmt -l ., go vet ./..., and go test ./... all passed with sandbox-local GOCACHE directories.
