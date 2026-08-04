---
id: 01KZ6ASH9PREHGTA67PP2HDPXG
kind: event
event_kind: commit
created: 2026-08-04T12:08:22Z
created_by: a-root
origin: agent
applied: false
---
2b0be80 skill: prefer ship over hand-run accept+integrate, and prune worktrees

Two things this session kept getting wrong, now written down.

accept moves the task file open/ -> done/, and that move is a
working-tree change somebody has to commit. Running accept and integrate
by hand and forgetting the commit made doctor report the task in two
status folders — three times today (251-testid, 221, 256). ship does
accept -> integrate -> commit the .dacli record -> push, and the third
step is the one that keeps getting dropped.

Worktree accumulation gets a real number instead of a caution: this repo
reached 103 checkouts and 2.4 GB before anyone looked, and the first
`worktree prune` took it to 6 and 44 MB. The dry-run classifies each
candidate as merged-into-trunk or run-finished, which is what makes it
checkable before deleting anything — a prune that only prints a count is
not one you can verify.

Also records that run as a metric: four agents were live at the time,
none appeared in the dry-run list, all four survived, suite green after.
role: root
