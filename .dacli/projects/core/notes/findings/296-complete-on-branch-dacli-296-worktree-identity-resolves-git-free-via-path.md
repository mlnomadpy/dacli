---
id: f-296-complete-on-branch-dacli-296-worktree-identity-resolves-git-free-via-path
kind: note
note_kind: finding
created: 2026-08-06T08:09:08Z
created_by: a-fixer-dh6km4
about: "[[296]]"
severity: major
---
# 296 complete on branch dacli/296: worktree identity resolves git-free via path, documented, regression-tested
All 3 acceptance criteria met on branch dacli/296-an-agent-token-minted-for-a-worktree-spawn-is-not-resolvable-from-inside-that.

(1) A token minted by spawn --worktree resolves from inside that worktree: internal/cli/worktree_identity_test.go TestWorktreeChildResolvesOwnIdentity spawns a real git worktree, mints an agent AFTER the worktree exists (so it is absent from the worktree's frozen .dacli git snapshot — the exact shadow bug), then runs `whoami` from inside the worktree with that token and asserts it resolves (grant: rw).

(2) The resolution rule is deterministic and documented, not dependent on which .dacli shadows which: internal/workspace/workspace.go mainWorktreeRoot now tries rootFromWorktreePath first — a pure path match on the `/.dacli/worktrees/` marker dacli itself always creates (WorktreePath), with the shared root being everything before that marker. No subprocess, so it can't silently fail the way `git rev-parse --path-format=absolute --git-common-dir` does (old git, or a sandbox withholding git) and drop resolution back onto the stale worktree snapshot. git rev-parse is now only a fallback for a hand-made worktree outside .dacli/worktrees/. Documented in docs/RUNTIMES.md new "Which workspace a worktree agent resolves" section under §19.

(3) internal/workspace/workspace_test.go TestFindRedirectsFromWorktreePathWithoutGit spawns a child into a worktree-shaped path with NO git repo at all and asserts Find still redirects to the shared root, not the physically-nested shadow.

PROOF: go build ./... clean, go vet ./... clean, gofmt -l clean, go test ./... all green. Red-green verified by hand: stashed workspace.go — TestFindRedirectsFromWorktreePathWithoutGit failed ("resolved the shadow, not the shared root") as expected, confirming the path-based fix (not the pre-existing git fallback) is what makes it pass. Note: TestWorktreeChildResolvesOwnIdentity still passes even with workspace.go stashed, because that test's environment has real git and the git rev-parse fallback still resolves correctly there — it is a genuine end-to-end regression test for criterion 3 (spawn into worktree, child resolves own identity), but does not by itself isolate the git-free code path; that isolation is covered by the workspace-level unit test in (3).

Owner: dacli accept 296 (task check is gated to root — I could not check the boxes myself).
