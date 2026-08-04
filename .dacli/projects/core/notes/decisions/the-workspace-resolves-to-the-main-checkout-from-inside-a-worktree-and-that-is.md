---
id: d-the-workspace-resolves-to-the-main-checkout-from-inside-a-worktree-and-that-is
kind: note
note_kind: decision
created: 2026-08-04T12:34:05Z
created_by: a-root
---
# The workspace resolves to the main checkout from inside a worktree, and that is intended
## Chose
The workspace resolves to the main checkout from inside a worktree, and that is intended
## Rejected
per-worktree .dacli, so each agent records onto its own branch
## Because
The workspace is the coordination substrate, and a per-branch store forks the backlog — which is exactly the failure behind tasks 251 (two branches allocate the same seq), the branch-local finding (root cannot dispatch a task waiting in an unmerged PR), and next/critical-path/burndown each computing a different project state per checkout. One shared store is what makes dacli agents, dacli next and dacli doctor describe reality rather than one branch's opinion of it. Concurrent writes are safe because task 247 gave the seq lock a real owner and a dead-holder-only steal. The cost is that an agent's record does not travel with its branch, which is a real loss and the right trade: the record is about the PROJECT, the branch is about the code. What is missing is not a different resolution but an explicit one — nothing in spawn or the skill says which .dacli a child is writing to, so every agent has to discover it.
