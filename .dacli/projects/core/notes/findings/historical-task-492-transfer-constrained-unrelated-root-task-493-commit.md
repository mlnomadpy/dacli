---
id: f-historical-task-492-transfer-constrained-unrelated-root-task-493-commit
kind: note
note_kind: finding
created: 2026-08-22T15:04:36Z
created_by: a-root
about: "[[494]]"
severity: major
---
# Historical task 492 transfer constrained unrelated root task 493 commit
While committing task 493 from its root-created worktree, dacli commit staged seven verified files then refused docs/MULTI_CLI.md and docs/RUNTIMES.md as outside claim [internal/store, internal/features/execution]. That claim came from .dacli/runs/01M0F8K1F3S6EKHHYGGQDENYMP/worktree-transfer.txt: a completed task 492 recovery into a different, now-pruned worktree. dacli agents reported no live agents. worktree reclaim correctly refused because the task 493 worktree has no spawned-agent ownership record. Manual step: preserve staged files, file this defect, then use the commit command's explicit audited --force once for the two reviewed documentation files.
