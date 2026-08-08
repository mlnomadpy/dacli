---
id: f-026-complete-on-branch-dacli-026-shadowing-commit-f1b2f5f-all-3-acceptance
kind: note
note_kind: finding
created: 2026-07-22T13:19:04Z
created_by: a-mzt5xcjgnm
about: [[026]]
severity: moderate
---
# 026 complete on branch dacli/026-...shadowing; commit f1b2f5f; all 3 acceptance criteria satisfied
Committed f1b2f5f by a-mzt5xcjgnm (maintainer) via git add + dacli commit --no-add, staging ONLY the 4 intended files: internal/workspace/workspace.go, internal/workspace/workspace_test.go, internal/features/execution/execution.go, and the 026 task file (agent file + finding notes excluded). ACCEPTANCE, all satisfied and verified in the diff: (1) workspace.Find redirects a linked-worktree cwd to the MAIN worktree .dacli via git rev-parse --git-common-dir (workspace.go:44-77 mainWorktreeRoot); new TestFindRedirectsFromLinkedWorktree proves both the redirect from the worktree and no-redirect from the main root, PASS 0.10s. (2) cmdSpawn --worktree no longer copies the child agent file into the worktree (removed the os.ReadFile/WriteFile block) and reads childEvents from the shared root w (execution.go), so a worktree child self-commits and self-checks against the ONE shared workspace. (3) committed on branch by an agent; go build ./... exit 0; go test ./internal/... all green incl. internal/workspace. Box-checking refused for non-owner (only a-root) — owner should verify and close via dacli task check/done + dacli merge --task 026.
