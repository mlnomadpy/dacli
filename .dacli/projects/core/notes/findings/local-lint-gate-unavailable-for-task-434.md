---
id: f-local-lint-gate-unavailable-for-task-434
kind: note
note_kind: finding
created: 2026-08-13T20:20:32Z
created_by: a-codex-maintainer-hh2s7h
about: "[[434]]"
severity: moderate
---
# Local lint gate unavailable for task 434
The mandated command golangci-lint run could not start because golangci-lint is not installed in PATH (exit 127). gofmt -l ., go vet ./..., go test ./..., and go test -v ./internal/scenarios all pass with GOCACHE=/private/tmp/dacli-434-gocache.
