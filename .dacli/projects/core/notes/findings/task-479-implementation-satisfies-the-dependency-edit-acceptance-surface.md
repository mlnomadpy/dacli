---
id: f-task-479-implementation-satisfies-the-dependency-edit-acceptance-surface
kind: note
note_kind: finding
created: 2026-08-20T08:34:29Z
created_by: a-maintainer-d7gr0n
about: "[[t-01M0CZANAQKP50AWEN2C6C8VXR]]"
severity: major
---
# Task 479 implementation satisfies the dependency-edit acceptance surface
Implemented documented task depend add/remove with FS/SS/FF/SF, stable/project-qualified resolution, atomic ambiguity/missing/self/type/cycle validation, audited direct events, read-only proposal plus owner sync, idempotent replay, adopted github mapping preservation, and shared CLI/MCP table help. Verified with compiled CLI add/show/remove/self-refusal; GOCACHE=/tmp/dacli-go-cache-479 go build ./..., go test ./..., go vet ./...; GOLANGCI_LINT_CACHE=/tmp/dacli-golangci-cache-479 golangci-lint run (0 issues); gofmt -l . empty; git diff --check clean. Acceptance checks remain owner-only.
