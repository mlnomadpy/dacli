---
id: f-task-404-lint-gate-unavailable-in-agent-environment
kind: note
note_kind: finding
created: 2026-08-13T09:49:59Z
created_by: a-codex-maintainer-2hqkmd
about: "[[404]]"
severity: moderate
---
# Task 404 lint gate unavailable in agent environment
Required formatting and go vet gates passed with GOCACHE=/private/tmp/dacli-404-gocache. The next command, 'golangci-lint run', could not execute because golangci-lint is not installed (zsh: command not found). Full go test is run separately; lint remains unverified here.
