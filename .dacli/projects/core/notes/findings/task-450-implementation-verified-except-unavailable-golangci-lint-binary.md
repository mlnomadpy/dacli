---
id: f-task-450-implementation-verified-except-unavailable-golangci-lint-binary
kind: note
note_kind: finding
created: 2026-08-14T00:06:44Z
created_by: a-maintainer-ry3evs
about: "[[450]]"
severity: minor
---
# Task 450 implementation verified except unavailable golangci-lint binary
GOCACHE=/tmp/dacli-450-gocache go build ./..., go test ./..., and go vet ./... passed; gofmt -l . was empty. golangci-lint run could not execute because golangci-lint is not installed (command not found). Mutation proof: changing ResolveLanding's legacy default to pr made internal/model TestResolveLandingPrecedence/legacy_default fail with got mode pr, want local.
