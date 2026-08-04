---
id: t-01KZ65YW5FX94R16J63C453J9K
kind: task
created: 2026-08-04T10:43:54Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# A role can declare grant rw and a read-only runtime, and nothing catches it
## So that
a spawn that cannot possibly write is refused at spawn time instead of burning a run
## Acceptance
- [ ] spawn refuses when the role grant is rw but the runtime has no write tool
- [ ] doctor reports every role whose grant and runtime capability disagree
- [ ] junior is either given a write-capable runtime or its grant is corrected
## Log
