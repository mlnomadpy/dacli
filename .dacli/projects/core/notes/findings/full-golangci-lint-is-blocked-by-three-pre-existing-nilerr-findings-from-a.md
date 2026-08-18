---
id: f-full-golangci-lint-is-blocked-by-three-pre-existing-nilerr-findings-from-a
kind: note
note_kind: finding
created: 2026-08-18T14:35:50Z
created_by: a-fixer-rmqgbs
about: "[[t-01M0AKHSFGWWSMDFCWCE9RYCGQ]]"
severity: moderate
---
# Full golangci-lint is blocked by three pre-existing nilerr findings from a removed sibling worktree
Running GOCACHE=/tmp/dacli-go-build-cache GOLANGCI_LINT_CACHE=/tmp/dacli-golangci-cache /Users/tahabsn/go/bin/golangci-lint run reported nilerr at internal/agentid/agentid_test.go:170 and internal/eventdisp/eventdisp.go:97,102, all attributed to removed core-460 worktree paths; no changed collab file is named.
