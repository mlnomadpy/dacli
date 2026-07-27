---
id: t-01KYHWEWPPN61AG13FABFNDAZB
kind: task
created: 2026-07-27T13:33:05Z
created_by: a-root
owner: a-root
priority: should
---
# merge conflict swallows SaveTask and MoveTask errors
## So that
a failed block-write does not leave a conflicted task marked runnable
## Acceptance
- [ ] lifecycle merge checks both write results before reporting blocked
## Log
