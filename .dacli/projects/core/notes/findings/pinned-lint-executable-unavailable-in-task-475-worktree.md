---
id: f-pinned-lint-executable-unavailable-in-task-475-worktree
kind: note
note_kind: finding
created: 2026-08-19T12:29:02Z
created_by: a-maintainer-pvm9jy
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: minor
---
# Pinned lint executable unavailable in task 475 worktree
Full verification on 2026-08-19 completed go build ./..., go test ./..., go vet ./..., and gofmt -l . clean with GOCACHE=/tmp/dacli-475-gocache. The final command could not run: zsh: command not found: golangci-lint; CONTRIBUTING.md pins v2.12.2. No network/install authority was available.
