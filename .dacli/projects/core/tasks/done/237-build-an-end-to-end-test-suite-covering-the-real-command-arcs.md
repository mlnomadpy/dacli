---
id: t-01KZ52TWY53635YFT2FRDCMEVD
kind: task
created: 2026-08-04T00:30:04Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 4, probable: 8, pessimistic: 16}"
---
# Build an end to end test suite covering the real command arcs
## So that
the arcs a user actually runs are proven to work together
## Acceptance
- [x] greenfield, adopt, sprint and land arcs each have an end to end test
- [x] the suite runs without network or spawning real agents
## Log
- 2026-08-04T09:35:59Z accepted by a-root
- 2026-08-04T09:35:59Z verified by `go test ./... >/dev/null` (exit 0)
- 2026-08-04T09:35:59Z completed by a-root
