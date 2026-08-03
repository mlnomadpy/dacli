---
id: t-01KZ4ECS7JM710Y878BPVMQH8V
kind: task
created: 2026-08-03T18:32:50Z
created_by: a-root
owner: a-root
priority: could
---
# Negative queue cursor panics instead of erroring
## So that
a hand-edited queue file cannot crash the CLI
## Acceptance
- [ ] an out-of-range cursor returns a usage error
## Log
