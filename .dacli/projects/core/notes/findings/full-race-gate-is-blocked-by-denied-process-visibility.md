---
id: f-full-race-gate-is-blocked-by-denied-process-visibility
kind: note
note_kind: finding
created: 2026-08-12T16:23:28Z
created_by: a-codex-maintainer-xm4nzv
about: "[[364]]"
severity: major
---
# Full race gate is blocked by denied process visibility
GOCACHE=/private/tmp/dacli-go-cache-364 go test -race ./... fails in pre-existing process-observation tests: internal/features/execution/execruntime_test.go:357 reads 0/164000 bytes, runstilllive_unix_test.go:36 cannot observe guardian identity, and internal/procmon tests see zero processes. Task-specific cross-process regression and affected package suites pass. golangci-lint is also unavailable (command not found).
