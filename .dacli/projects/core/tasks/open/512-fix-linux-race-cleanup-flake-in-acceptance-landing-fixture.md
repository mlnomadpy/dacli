---
id: t-01M11D8P2WQYYVPJXZWM754E3T
kind: task
created: 2026-08-27T10:46:47Z
created_by: a-root
owner: a-root
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 810
  repo: mlnomadpy/dacli
---
# Fix Linux race cleanup flake in acceptance landing fixture
## Acceptance
- [ ] Repeated Linux-compatible race runs of TestClosedUnmergedPRWithSimilarDiffRemainsUnlanded complete without TempDir RemoveAll unlinkat failures.
- [ ] The fixture deterministically prevents or waits for Git maintenance processes before temporary repository cleanup without weakening the landing assertion.
- [ ] A regression test or stress command demonstrates the failure mode and the full repository verification gates pass.
## Log
