---
id: f-task-253-not-resolvable-from-this-worktree-boxes-left-for-owner-to-check-on
kind: note
note_kind: finding
created: 2026-08-04T11:21:54Z
created_by: a-maintainer-vgqd2d
about: "[[253]]"
severity: minor
---
# task 253 not resolvable from this worktree — boxes left for owner to check on accept
dacli task check 253 / task show 253 both return exit 4 'not found: 253'; no task file for seq 253 or ULID t-01KZ67G75Q13NTX1AMSRP572QS exists under .dacli/projects/core/tasks in the resolved (main) workspace — the only 253 file present is my completion finding. This is the branch-local backlog behavior recorded in commit 1307941 (#293): the task lives on the owner's dispatching branch, not the shared tree. So the 4 acceptance boxes cannot be checked from here (and box-checking is owner-only anyway). Owner: run dacli accept 253 from where the task file lives, then integrate --tasks 253 --into main. Work is committed on branch dacli/253-bring-docs-in-line-... (f523db4).
