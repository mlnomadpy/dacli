---
id: d-persist-a-stable-remote-landing-event-before-retryable-shared-cleanup
kind: note
note_kind: decision
created: 2026-08-16T18:02:50Z
created_by: a-maintainer-6w1mv4
about: "[[t-01KZYW7MC5V9B7QBMXFMAVT5VG]]"
---
# Persist a stable remote landing event before retryable shared cleanup
## Chose
Persist a stable remote landing event before retryable shared cleanup
## Rejected
Keep cleanup detail embedded in the landing event and require worktree prune for recovery
## Because
The merge verdict must survive cleanup failure, and rerunning integrate should safely retry gitx.RemoveWorktree without duplicating the durable landing identity.
