---
id: f-repository-lint-is-blocked-by-stale-core-460-worktree-diagnostics
kind: note
note_kind: finding
created: 2026-08-18T15:14:21Z
created_by: a-maintainer-dgyp5f
about: "[[t-01M0AEG5F23TRH6BAR9HT38ZP1]]"
severity: moderate
---
# Repository lint is blocked by stale core-460 worktree diagnostics
Pinned golangci-lint v2.12.2 scans deleted sibling core-460 paths and exits with three unrelated nilerr diagnostics in internal/agentid/agentid_test.go and internal/eventdisp/eventdisp.go. Focused lint on ./internal/features/orchestration/... passes with 0 issues; go build ./..., go vet ./..., go test ./..., and orchestration -race pass.
