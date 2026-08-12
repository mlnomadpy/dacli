---
id: t-01KZV2X46ATDJJNSJKB8FZWCY3
kind: task
created: 2026-08-12T13:34:34Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
depends_on: [370]
github:
  issue: 466
  repo: mlnomadpy/dacli
---
# Honor explicit loop implementer roles over automatic cost routing
## So that
An operator-supplied --impl-role selects the requested runtime and method predictably
## Acceptance
- [ ] loop --impl-role R spawns every build task with R unless a phase gate explicitly refuses that role
- [ ] Cheapest-capable per-task routing remains the default only when --impl-role is omitted
- [ ] Dry-run output and loop advise explain whether the role came from an explicit override, phase routing, or automatic cost routing
- [ ] A regression roster proves an explicit backend role is not replaced by a cheaper frontend role for a task mentioning docs/RUNTIMES.md
- [ ] go test -race ./... passes
## Log
