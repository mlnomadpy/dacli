---
id: d-probe-the-task-branch-s-existing-merged-pr-before-any-push
kind: note
note_kind: decision
created: 2026-08-17T16:08:33Z
created_by: a-maintainer-w1qy51
about: "[[t-01KZYW7MC5V9B7QBMXFMAVT5VG]]"
---
# Probe the task branch's existing merged PR before any push
## Chose
Probe the task branch's existing merged PR before any push
## Rejected
Push the local branch first and recover only after merge or PR creation fails
## Because
GitHub may already have merged the PR and deleted its head; pushing recreates stale remote state and PR creation then fails with no commits, while a confirmed merge can immediately use the durable landing and shared cleanup path.
