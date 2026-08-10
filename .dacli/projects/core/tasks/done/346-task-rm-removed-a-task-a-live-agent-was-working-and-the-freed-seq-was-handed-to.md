---
id: t-01KZPRKZF0TB24JXJR8TQR44BA
kind: task
created: 2026-08-10T21:17:51Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# task rm removed a task a live agent was working, and the freed seq was handed to a different task
## Acceptance
- [x] removing a task whose run record names it and whose agent process is alive is refused, and the refusal names the agent and how to stop it
- [x] --force does not override a live claimant, because the alternative is a run that cannot be made correct
- [x] removal still works once no live agent holds it, so the guard is about liveness rather than about ever having been claimed
## Log
- 2026-08-10T21:18:33Z accepted by a-root
- 2026-08-10T21:18:33Z verified by `go test ./internal/features/planning/ ./internal/store/` (exit 0)
- 2026-08-10T21:18:33Z deliverable: no dacli/346-task-rm-removed-a-task-a-live-agent-was-working-and-the-freed-seq-was-handed-to branch — nothing to check against sprint/7
- 2026-08-10T21:18:33Z completed by a-root
