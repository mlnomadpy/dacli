---
id: m-worktree-reclamation-103-checkouts-and-2-4-gb-down-to-6-and-44-mb
kind: note
note_kind: metric
created: 2026-08-04T12:07:43Z
created_by: a-root
---
# Worktree reclamation: 103 checkouts and 2.4 GB down to 6 and 44 MB
First run of `dacli worktree prune` (task 252) on this workspace, immediately after it landed.

Before: 103 worktrees, 2.4 GB under .dacli/worktrees — one per task ever spawned with --worktree since the project started, none ever reclaimed.
After: 6 worktrees, 44 MB. 97 reclaimed in one pass.

Correctness checks that mattered more than the number:
- The four agents live at the time (tasks 220, 183, 223, 224) were NOT in the dry-run list and were still running afterwards.
- `go build ./... && go test ./... -count=1` green after the prune.
- The dry-run classified each candidate by reason — 'merged into main' or 'run finished' — so the decision was inspectable before anything was deleted. That is what made it safe to run at all; a prune that just prints a count is not one you can check.

Worth knowing for the loop: it calls this each cycle, so the steady state is bounded rather than the 2.4 GB drift that accumulated here over roughly a week of waves.
