---
id: f-a-root-s-unmerged-dacli-302-branch-e850f9a-duplicated-this-task-via-the-still
kind: note
note_kind: finding
created: 2026-08-05T13:27:45Z
created_by: a-fixer-zrcq2v
about: "[[302]]"
severity: moderate
---
# a-root's unmerged dacli/302 branch (e850f9a) duplicated this task via the still-open cross-branch seq collision (dacli-251)
origin/dacli/302-spawn-worktree-... has one commit (e850f9a, a-root, 2026-08-05T05:19:20-07:00, unmerged) that independently created a SECOND task file at the identical path .dacli/projects/core/tasks/open/302-spawn-worktree-....md with a different id (t-01KZ8XERJG6ZWZ8PJTQRNWDX0S, created 12:13:01Z) than the one I claimed (t-01KZ7EQGGWV8JS8J346KD2XCFQ, created 22:36:24Z the prior day). dacli-251's gitTaskSeqCeiling scans 'git log --all' from the creating checkout to avoid exactly this, but a-root's checkout apparently didn't have my task's ref fetched/reachable at creation time, so the ceiling scan missed it -- the collision guard is only as good as what --all can see locally. That branch's commit (gitx-style trunk-contamination DETECTION via 'dacli doctor', not integrated into spawn) never merged and its promised follow-up ('filed separately') was never actually filed -- no task exists for relocating worktrees outside the main checkout. Recommend: close/abandon the orphaned origin branch and duplicate task file once reviewed, and treat the 'move worktrees outside main' idea as unfiled backlog, not an existing task.
