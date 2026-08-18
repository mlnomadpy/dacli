---
id: t-01M0AGCX29Q047FZHKNG3YV0WC
kind: task
created: 2026-08-18T13:18:59Z
created_by: a-root
owner: a-root
github:
  issue: 694
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Add an audited recovery path for worktrees owned by failed pre-task agents
## Context
Adopted from GitHub issue #694.

## Symptom

A correction spawn for task 468 failed before reading the prompt (`Codex app-server: Operation not permitted`). A retry on Claude also failed before task execution (`Not logged in`). Each spawn rewrote the worktree owner to its newly minted child. The child token is intentionally one-time and unavailable after the failed process exits. `dacli commit` then refuses root and later agents with exit 3: restore the owning child token. There is no supported way to do that, even when the child is terminal, retired, produced no events, and never touched the checkout.

## Reproduction

1. Spawn a write task with `--worktree` using a runtime that exits during initialization.
2. Finalize it with `dacli wait`; confirm the child is retired and the run is terminal/no-visible-result.
3. Make or retain a verified change in that worktree.
4. As root or a new correction agent, run `dacli commit`.
5. Observe exit 3 naming the failed child and an impossible remedy (restore a one-time token).

## Suspected cause

`internal/features/vcs.agentWorktreeOwner` chooses the newest run whose `worktree.txt` points at the checkout and `cmdCommit` requires exact child identity without considering terminal state, explicit root recovery, or an ownership-transfer event. The latest spawn can therefore permanently seize a worktree even when it never started task work.

## Risk

Runtime startup/authentication failures turn a recoverable code state into an uncommittable worktree. Repeated correction spawns make ownership less recoverable, encourage raw Git bypasses, and prevent loop self-healing.

## Manual workaround

Create a fresh root-owned worktree/branch from the verified commits, reapply the remaining patch, rerun verification, and open a PR from that branch. Do not recover or impersonate the one-time child credential.

## Design

Add an audited, fail-closed ownership transfer/reclaim operation. Permit root recovery only when every owning run for that worktree is terminal, no process is live, the current diff/claim is shown, and the transfer is recorded. Preserve the original run/agent attribution; do not mutate historical identity. Refuse while any owner is live or state is unreadable.

## Acceptance criteria

- A public regression reproduces a runtime that exits before task execution and proves the owner can explicitly reclaim or transfer the worktree without the lost token.
- Recovery refuses with exit 3 while the owning process is live or run state is unreadable.
- The command previews exact worktree, branch, prior owner/run, dirty paths, and claims before mutation.
- The transfer/reclaim is durable and auditable; historical agent/run records remain unchanged.
- A subsequent governed `dacli commit` succeeds under the new identity and enforces the transferred claim.
- Repeated failed correction spawns do not make the worktree permanently unrecoverable.
- Mutation evidence, focused vcs/execution tests, race tests, and `go test ./...` pass.

## Acceptance
- [ ] A public regression reproduces a runtime that exits before task execution and proves the owner can explicitly reclaim or transfer the worktree without the lost token.
- [ ] Recovery refuses with exit 3 while the owning process is live or run state is unreadable.
- [ ] The command previews exact worktree, branch, prior owner/run, dirty paths, and claims before mutation.
- [ ] The transfer/reclaim is durable and auditable; historical agent/run records remain unchanged.
- [ ] A subsequent governed `dacli commit` succeeds under the new identity and enforces the transferred claim.
- [ ] Repeated failed correction spawns do not make the worktree permanently unrecoverable.
- [ ] Mutation evidence, focused vcs/execution tests, race tests, and `go test ./...` pass.
## Log
