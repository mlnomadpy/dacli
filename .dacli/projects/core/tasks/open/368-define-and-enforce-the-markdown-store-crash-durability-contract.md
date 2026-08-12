---
id: t-01KZV16EPCRG5DXDSVW08TSSH6
kind: task
created: 2026-08-12T13:04:43Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
depends_on: [364]
github:
  issue: 461
  repo: mlnomadpy/dacli
---
# Define and enforce the markdown store crash-durability contract
## So that
Event and task persistence has an explicit, testable guarantee across process and machine crashes
## Acceptance
- [ ] The storage contract explicitly distinguishes atomic rename from power-loss durability
- [ ] If power-loss durability is promised, write paths sync file data and the containing directory in the required order
- [ ] Focused tests or injected filesystem operations verify the promised ordering and error propagation
- [ ] All task, event, and runtime-file writers conform to the documented contract
- [ ] go test -race ./... passes
## Log
