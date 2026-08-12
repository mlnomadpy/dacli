---
id: f-golangci-lint-unavailable-in-task-sandbox
kind: note
note_kind: finding
created: 2026-08-12T19:29:04Z
created_by: a-codex-maintainer-djpe71
about: "[[397]]"
severity: minor
---
# golangci-lint unavailable in task sandbox
The required golangci-lint run could not execute because zsh reported command not found. gofmt -l ., go vet ./..., focused tests, and go test ./... all passed; lint remains unverified in this environment.
