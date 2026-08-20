---
id: f-pinned-lint-check-unavailable-in-task-environment
kind: note
note_kind: finding
created: 2026-08-19T15:24:02Z
created_by: a-maintainer-3necr2
about: "[[t-01M0CX03Q4A1BM8JD9YQBCNGV0]]"
severity: minor
---
# Pinned lint check unavailable in task environment
The exact build/test/vet/gofmt checks pass with GOCACHE=/tmp/dacli-go-cache-477. The prescribed golangci-lint run could not execute because golangci-lint is not installed (zsh: command not found); no lint result is claimed.
