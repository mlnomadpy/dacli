---
id: f-pinned-golangci-lint-is-unavailable-in-this-run
kind: note
note_kind: finding
created: 2026-08-13T16:15:27Z
created_by: a-codex-maintainer-sg0bxk
about: "[[425]]"
severity: moderate
---
# Pinned golangci-lint is unavailable in this run
The required golangci-lint run could not execute because golangci-lint is not installed on PATH. gofmt, go vet, focused tests, and the full Go suite were run with a sandbox-writable GOCACHE; lint remains explicitly unverified.
