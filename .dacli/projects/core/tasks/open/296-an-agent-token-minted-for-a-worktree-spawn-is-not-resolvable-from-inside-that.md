---
id: t-01KZ70FTXVYA30SV04A0BY10P1
kind: task
created: 2026-08-04T18:27:33Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 3, pessimistic: 6}"
---
# An agent token minted for a worktree spawn is not resolvable from inside that worktree
## So that
an agent in a worktree can use dacli at all, instead of falling back to raw git and reporting upstream
## Acceptance
- [ ] a token minted by spawn --worktree resolves from the worktree the agent is started in
- [ ] the workspace a command resolves to from inside a worktree is deterministic and documented, not dependent on which .dacli shadows which
- [ ] a test spawns into a worktree and has the child resolve its own identity
## Log
- 2026-08-04T20:41:33Z claimed by a-maintainer-tcn4vf
