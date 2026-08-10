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
- [ ] both functions can report a read failure to their caller instead of returning an empty slice for two different facts
- [ ] the WIP gate does not read an unreadable runs directory as 'nobody is working'
- [ ] a test asserts an unreadable runs directory surfaces as an error rather than an empty result
## Log
