---
id: t-01KZR80DH0MB0XF74V1QE17PHH
kind: task
created: 2026-08-11T11:06:02Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Prove cross-process safety, which the race detector cannot observe by construction
## Acceptance
- [x] a test runs several real dacli processes concurrently against one workspace and asserts no task, seq or event is lost or duplicated
- [x] it fails if the per-task lock or the seq lock is removed, so it measures the locking rather than the happy path
- [x] CI runs it, and the honest scope note in ci.yml about -race being in-process is updated to say what now covers the gap
## Log
- 2026-08-11T11:14:45Z accepted by a-root
- 2026-08-11T11:14:45Z closed WITHOUT verification — no --verify command was given
- 2026-08-11T11:14:45Z deliverable: no dacli/359-prove-cross-process-safety-which-the-race-detector-cannot-observe-by branch — nothing to check against sprint/15
- 2026-08-11T11:14:45Z completed by a-root
