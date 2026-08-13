---
id: f-golangci-lint-unavailable-for-task-434-local-verification
kind: note
note_kind: finding
created: 2026-08-13T20:02:47Z
created_by: a-codex-maintainer-grz3zz
about: "[[434]]"
severity: moderate
---
# golangci-lint unavailable for task 434 local verification
The required command golangci-lint run could not execute because golangci-lint is not installed (zsh: command not found). gofmt -l ., go vet ./..., go test ./.github/workflows, focused scenario tests, mutation tests, and go test ./... pass.
