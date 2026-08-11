---
id: t-01KZRT0CQATNPYPP78NED33Y92
kind: task
created: 2026-08-11T16:20:35Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Sweep for candidate-list loops that return a negative verdict from inside the loop
## Acceptance
- [ ] Every 'for ... range' over a candidate list (refs, paths, ceilings, runtimes) in internal/ is inspected for a return of a NEGATIVE or default result from inside the loop body, which makes later candidates unreachable
- [ ] Each site is either confirmed correct with a comment stating why the first responding candidate is authoritative, or fixed so the loop exhausts its candidates before concluding
- [ ] A finding records the sites inspected and the verdicts, so a later reader does not repeat the sweep
- [ ] Any fix carries a test that fails on the unfixed code, in the style of TestLandingOfRefSeesLocalTrunkWhenOriginIsStale
## Log
