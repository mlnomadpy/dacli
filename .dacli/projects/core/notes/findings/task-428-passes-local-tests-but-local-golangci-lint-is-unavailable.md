---
id: f-task-428-passes-local-tests-but-local-golangci-lint-is-unavailable
kind: note
note_kind: finding
created: 2026-08-13T19:23:07Z
created_by: a-codex-maintainer-xytv4d
about: "[[428]]"
severity: moderate
---
# Task 428 passes local tests but local golangci-lint is unavailable
gofmt -l . produced no output, go vet ./... passed, focused workflow contract and go test ./... passed. golangci-lint run could not execute because the binary is not installed; the PR's pinned lint CI job must supply that evidence.
