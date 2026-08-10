---
id: t-01KZPFB34G7B4YMF0JNN7VPE5P
kind: task
created: 2026-08-10T18:35:43Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# liveAgents and sweepFinishedDetached cannot distinguish an unreadable runs directory from no runs
## Acceptance
- [x] both functions can report a read failure to their caller instead of returning an empty slice for two different facts
- [x] the WIP gate does not read an unreadable runs directory as 'nobody is working'
- [x] a test asserts an unreadable runs directory surfaces as an error rather than an empty result
## Log
- 2026-08-10T19:12:34Z claimed by a-fixer-7s538v
- 2026-08-10T19:27:32Z accepted by a-root (applied 1 proposal(s))
- 2026-08-10T19:27:32Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T19:27:32Z completed by a-root
- 2026-08-10T19:27:33Z deliverable: dacli/337-liveagents-and-sweepfinisheddetached-cannot-distinguish-an-unreadable-runs exists but is NOT in trunk — closed anyway
- 2026-08-10T19:35:22Z accepted by a-root
- 2026-08-10T19:35:22Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T19:35:22Z deliverable: no dacli/337-liveagents-and-sweepfinisheddetached-cannot-distinguish-an-unreadable-runs branch — nothing to check against trunk
- 2026-08-10T19:35:22Z completed by a-root
