---
id: f-record-stage-is-genuinely-good-commit-is-honest-about-no-ops-unresolved-role
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-reviewer-664htb
about: "[[t-01KZ6S9FVG533ABEVXZBZZ7SC3]]"
source_event: 01KZ6SV708N5HBN6MQ4RVGHY6K
---
# record stage is genuinely good: commit is honest about no-ops, unresolved role, and worktree crumb routing -- stated per the audit's 'say when a stage is good' rule, not a defect
Stage: RECORD -- NOT A DEFECT, an affirmation (acceptance asks to state a good stage rather than pad). dacli commit does not fake success: nothing staged returns exit 2 'nothing staged to commit' (vcs.go:105-108), never a 'committed' line on a no-op. Unresolved role degrades LOUDLY (stderr warning naming the likely cause -- stale worktree .dacli/agents/) rather than silently dropping the Dacli-Role trailer (vcs.go:147-150). Commits on main/master are refused (vcs.go:95-98). The event crumb is written via w (workspace.Find redirect to the shared main root, vcs.go:178) not ctx.Cwd, so a worktree child's commit crumb reaches the shared store, not its stale worktree .dacli (matches task 026). And dacli ask (collab.go:109-120, dacli 197) now reports honestly that a read-only child's block is a pending event applied on owner sync, instead of the old lie that the task was 'blocked until answered'. The one caveat (not a new finding): the worktree-redirect guarantee is contingent on 'git rev-parse --git-common-dir' resolving (workspace.go); if git cannot resolve it, Find falls back to the local stale .dacli -- already the subject of the worktree-shadows finding.
