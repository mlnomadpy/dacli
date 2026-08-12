---
id: f-pinned-golangci-lint-unavailable-in-task-sandbox
kind: note
note_kind: finding
created: 2026-08-12T19:25:07Z
created_by: a-codex-maintainer-hyzqzv
about: "[[373]]"
severity: minor
---
# Pinned golangci-lint unavailable in task sandbox
The required verification command reached golangci-lint run after gofmt -l . and go vet ./... succeeded, then zsh returned command not found (exit 127). Full normal and race test suites are being run separately; lint remains unverified unless a local executable is found.
