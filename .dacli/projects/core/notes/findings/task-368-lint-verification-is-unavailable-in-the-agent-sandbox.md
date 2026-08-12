---
id: f-task-368-lint-verification-is-unavailable-in-the-agent-sandbox
kind: note
note_kind: finding
created: 2026-08-12T19:27:43Z
created_by: a-codex-maintainer-zf35yj
about: "[[368]]"
severity: minor
---
# Task 368 lint verification is unavailable in the agent sandbox
The required golangci-lint run could not start because golangci-lint is not installed (zsh: command not found). gofmt -l ., go vet ./..., go test ./..., and go test -race ./... all passed with GOCACHE=/private/tmp/dacli-368-go-cache.
