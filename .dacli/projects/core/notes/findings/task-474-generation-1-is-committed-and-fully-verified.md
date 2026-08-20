---
id: f-task-474-generation-1-is-committed-and-fully-verified
kind: note
note_kind: finding
created: 2026-08-19T11:41:24Z
created_by: a-fixer-4y3mj5
about: "[[t-01M0AKHSFGWWSMDFCWCE9RYCGQ]]"
severity: major
---
# Task 474 generation 1 is committed and fully verified
Commit ca7d196 on branch dacli/474-allow-root-to-resolve-stale-proposals-owned-by-the-loop-anchor allows rw root to dismiss a loop-anchor proposal from a known unretired actor only when a valid completed run record proves terminal lifecycle; live, unknown, never-run, ordinary-owner, read-only, and unresolved/corrupt cases remain fail-closed. Mutation restoring the retired-only guard failed TestRootDismissesFinishedUnretiredProposalOnLoopAnchor at collab_test.go:460 with refused-unrelated. Verified: gofmt -l .; go vet ./...; GOCACHE=/tmp/dacli-474-lint-go-cache GOLANGCI_LINT_CACHE=/tmp/dacli-474-golangci-cache /Users/tahabsn/go/bin/golangci-lint run (0 issues); go test ./...; go test -race ./internal/features/collab; go test ./internal/eventlog. PR-first is off; owner should accept/integrate this branch.
