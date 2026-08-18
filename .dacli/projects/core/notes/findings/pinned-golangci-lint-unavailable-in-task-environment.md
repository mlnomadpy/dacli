---
id: f-pinned-golangci-lint-unavailable-in-task-environment
kind: note
note_kind: finding
created: 2026-08-18T14:25:24Z
created_by: a-maintainer-zppm9n
about: "[[t-01M0AEG5AQPVJTH41MJNFRGSSX]]"
severity: minor
---
# Pinned golangci-lint unavailable in task environment
The required golangci-lint run could not execute because golangci-lint is not installed or on PATH. go build ./..., go test ./..., go vet ./..., gofmt -l ., and go test -race ./internal/features/execution all pass with writable /tmp Go caches.
