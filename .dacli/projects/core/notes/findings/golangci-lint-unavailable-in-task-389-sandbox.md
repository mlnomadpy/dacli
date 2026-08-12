---
id: f-golangci-lint-unavailable-in-task-389-sandbox
kind: note
note_kind: finding
created: 2026-08-12T19:06:19Z
created_by: a-codex-maintainer-j94tjr
about: "[[389]]"
severity: minor
---
# golangci-lint unavailable in task 389 sandbox
The required  invocation was attempted after implementation but the binary is not installed (). gofmt -l ., go vet ./..., go test ./..., and go test -race ./internal/features/orchestration all passed with GOCACHE under /private/tmp.
