---
id: 01KZ66BYNPBEN6AE4DMTJ721DD
kind: event
event_kind: commit
created: 2026-08-04T10:51:03Z
created_by: a-root
origin: agent
applied: false
---
fba104f record: the backlog is branch-local, so a task cannot be dispatched until its PR merges

`dacli spawn --task 250` returned 'not found: 250' from main thirty
seconds after `task add` created it — the record was committed on the
branch behind PR #292, and main's tree has no such file.

The seq collision filed as 251 is the visible symptom; this is its
shape. Tasks, notes, risks and lessons are files in the tree, so the
backlog forks with the branch: root cannot dispatch a task waiting in
an unmerged PR, a worktree agent sees the backlog as of its branch
point, and next/critical-path/burndown each report a different project
state per tree with none of them wrong.

Filing straight onto main sidesteps it, which is what the loop
effectively does — but that is a convention holding the invariant, not
a mechanism.
role: root
