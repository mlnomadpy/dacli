---
id: f-the-whole-backlog-is-branch-local-so-root-cannot-dispatch-a-task-it-filed-one
kind: note
note_kind: finding
created: 2026-08-04T10:50:46Z
created_by: a-root
---
# The whole backlog is branch-local, so root cannot dispatch a task it filed one commit ago
`dacli spawn --task 250` returned `not found: 250` from main, thirty seconds after `dacli task add` had created it. The task file was committed on the branch behind PR #292 and main's tree has no such file, so as far as every command run from main is concerned the task does not exist.

The seq collision filed as task 251 is the visible symptom; this is the shape of it. Task records, notes, risks and lessons all live as files in the tree, so the backlog forks with the branch. Concretely:

- Root cannot spawn an agent on a task that is waiting in an unmerged PR — the whole point of filing it was to dispatch it.
- An agent in a `--worktree` checkout sees the backlog as of its branch point, so anything filed since is invisible to it, including a task saying 'stop, this is being handled elsewhere'.
- `dacli next`, `critical-path` and `burndown` all compute over one tree, so each branch reports a different project state and none of them is wrong.
- Reviews queue behind merges: with nine PRs open and merging gated, the backlog visible from main is now several waves stale.

Filing tasks straight onto main sidesteps it and is what the loop effectively does, but that is a convention holding the invariant, not a mechanism. Anything wanting the record reviewable before it lands hits this immediately.

Recorded from the main tree, on its own branch, which is the joke.
