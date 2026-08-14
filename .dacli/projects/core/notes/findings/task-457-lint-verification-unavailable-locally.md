---
id: f-task-457-lint-verification-unavailable-locally
kind: note
note_kind: finding
created: 2026-08-14T09:57:18Z
created_by: a-maintainer-z7dcp1
about: "[[t-01KZZSD1K4YT88J0YYB5ZPD75R]]"
severity: moderate
---
# Task 457 lint verification unavailable locally
golangci-lint is not installed in this environment (zsh: command not found). gofmt -l ., go build ./..., go vet ./..., and go test ./... passed with GOCACHE and GOMODCACHE under /tmp because default user cache paths are sandbox-read-only.
