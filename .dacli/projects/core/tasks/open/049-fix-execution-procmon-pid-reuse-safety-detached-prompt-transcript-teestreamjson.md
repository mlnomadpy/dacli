---
id: t-01KY59YVZWW6NB52VRS3CJF7M1
kind: task
created: 2026-07-22T16:18:52Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# FIX execution/procmon: PID reuse safety, detached prompt+transcript, teeStreamJSON, CPU label
## Acceptance
- [ ] PID/PGID reuse guarded: a stale proc.txt cannot make a dead run resurface as live or cause KillTree to signal an unrelated process group (validate start-time or process identity)
- [ ] stdin-mode --detach no longer truncates/drops the prompt (parent must not exit before the stdin copy completes); detached stream-json writes READABLE text to transcript.log so logs -f/--tail work
- [ ] teeStreamJSON checks sc.Err() and handles over-long lines so token usage is not silently lost; agents %CPU is labelled as ps lifetime-average (or computed as current), not mislabelled
- [ ] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T16:19:09Z claimed by a-1syt6ccpg3
