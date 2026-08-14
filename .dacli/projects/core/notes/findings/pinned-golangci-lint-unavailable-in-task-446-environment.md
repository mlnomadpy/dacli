---
id: f-pinned-golangci-lint-unavailable-in-task-446-environment
kind: note
note_kind: finding
created: 2026-08-14T01:52:26Z
created_by: a-maintainer-11wpsr
about: "[[446]]"
severity: minor
---
# Pinned golangci-lint unavailable in task 446 environment
CONTRIBUTING.md pins golangci-lint v2.12.2, but running 'golangci-lint run' returned zsh: command not found. gofmt -l ., go vet ./..., go build ./..., and go test ./... completed successfully; lint remains unverified locally and must run in CI.
