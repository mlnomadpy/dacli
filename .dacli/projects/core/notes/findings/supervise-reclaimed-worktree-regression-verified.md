---
id: f-supervise-reclaimed-worktree-regression-verified
kind: note
note_kind: finding
created: 2026-08-28T09:35:46Z
created_by: a-fixer-5xjt19
about: "[[t-01M13V42QDZE7CKYDFWYVB5YG5]]"
severity: minor
---
# Supervise reclaimed-worktree regression verified
internal/features/execution/spawn_worktree_test.go: TestSuperviseCorrectionResumesRootReclaimedTaskWorktreeAcrossTurns creates a terminal child plus audited root transfer, runs two supervise turns, and verifies both execute and are recorded in the task worktree. Mutation: changing supervise execRuntime(workDir, ...) back to w.Root makes the test fail at spawn_worktree_test.go:157 because supervise-cwds.txt is absent from the task worktree. Focused execution/VCS tests and gofmt, go vet, go test ./... passed; focused golangci-lint execution passed (workspace-wide lint is polluted by stale sibling-worktree paths).
