---
id: t-01KZPWRYAZ4TBWT1C3G4JBTTW5
kind: task
created: 2026-08-10T22:30:28Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Build an end-to-end fixture repository dacli plans, implements, reviews and ships unattended
## Acceptance
- [x] a fixture repo is checked in that a single documented command drives from empty to shipped without manual intervention
- [x] the run is offline and deterministic: a stub runtime stands in for the agent, so it can execute in CI
- [x] it asserts the OUTCOME — the code exists, the tests pass, the task closed, the branch landed — not that commands were called
- [x] it fails if any step silently produces nothing, which is the failure class this repo keeps hitting
## Log
- 2026-08-11T10:05:51Z accepted by a-root
- 2026-08-11T10:05:51Z verified by `go test ./internal/cli/ -run TestE2EFixtureRepo` (exit 0)
- 2026-08-11T10:05:51Z deliverable: no dacli/351-build-an-end-to-end-fixture-repository-dacli-plans-implements-reviews-and-ships branch — nothing to check against sprint/13
- 2026-08-11T10:05:51Z completed by a-root
