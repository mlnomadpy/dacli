---
id: f-task-399-passes-repository-tests-but-lint-binary-is-unavailable
kind: note
note_kind: finding
created: 2026-08-12T20:13:13Z
created_by: a-codex-maintainer-csf6ta
about: "[[399]]"
severity: minor
---
# Task 399 passes repository tests but lint binary is unavailable
gofmt -l . was empty, go vet ./... passed, and go test ./... passed. golangci-lint run could not execute because golangci-lint is not installed in PATH; lint remains unverified locally.
