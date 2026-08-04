---
id: t-01KZ70373GFPDZKSM1ZF4BHVG2
kind: task
created: 2026-08-04T18:20:39Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# pr --auto exits 0 with a stderr note when auto-merge could not be set, so the caller believes it landed
## So that
an agent that queued an auto-merge and got exit 0 is right to believe the PR will land
## Acceptance
- [ ] a failure to queue auto-merge is visible in the exit status or in a recorded outcome, not only on stderr
- [ ] the loop's land phase distinguishes queued from not-queued rather than treating both as done
## Log
