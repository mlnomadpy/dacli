---
id: t-01KZP9495VB59MMNWTYFXJF3C3
kind: task
created: 2026-08-10T16:47:08Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Cover the unlanded-close refusal end to end: acceptOne and acceptAll under --require-verify
## Acceptance
- [ ] a test builds a task branch with commits NOT merged to trunk, drives acceptOne with requireVerify=true and allowUnlanded=false, and asserts exit 3 plus the task staying open
- [ ] the same coverage exists for acceptAll (acceptance.go:260-266), which is the batch path the loop uses
- [ ] a sibling test with allowUnlanded=true still closes, proving the flag rather than a blanket refusal
- [ ] both tests fail when 'if requireVerify {' at acceptance.go:148 is changed to 'if false {'
## Log
