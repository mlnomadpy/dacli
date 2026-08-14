---
id: t-01KZZVWMD0KYAPDN9QMQDK1GF3
kind: task
created: 2026-08-14T10:08:10Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 673
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Preserve the claimed task worktree when resuming Codex agents
## Context
Adopted from GitHub issue #673.

In azettaai/perio task 044, dacli spawn --task 044 --role fixer --runtime codex-terra-rw --worktree --detach --pr correctly created .dacli/worktrees/periodica-044-..., but dacli wait finalized run 01KZZTFZ6GH2RMA9PP41P5AG8A after 59s as 'no visible result' while the Codex process later completed and committed 5313f08. A resume invoked from the task worktree with dacli spawn --task 044 --role frontend-associate --runtime codex-terra-rw --detach --pr launched with cwd=/Users/tahabsn/Documents/GitHub/perio (dev), not the caller's task worktree, forcing the agent to restore its accidental edit and stop. dacli verify showed the same late-result race twice: it returned no-verdict, then transcripts recorded confirmed findings after exact-tree gates passed. Expected: preserve/resolve the task worktree cwd for resumed runs and wait for the runtime's final result/finding before finalizing detached or verify outcomes.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] A regression starts from an existing task worktree and spawns a follow-up agent for the same task without `--worktree`; the runtime cwd resolves to that task worktree, not the workspace main checkout.
- [ ] If more than one candidate worktree or an unsafe mismatch exists, spawn refuses with exit 3 and names the explicit recovery command.
- [ ] The resumed agent's edits and `dacli commit` land only on the task branch and preserve the original task/worktree attribution.
- [ ] Spawns issued from the main checkout retain current behavior and do not inherit an unrelated task worktree.
- [ ] Mutation evidence, focused execution/worktree tests, and `go test ./...` pass.
## Log
