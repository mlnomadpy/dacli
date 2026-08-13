---
id: t-01KZX7PXQBEVM1M0N2BKWYD4RK
kind: task
created: 2026-08-13T09:37:03Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
depends_on: "[403:FS, 404:FS]"
github:
  issue: 550
  repo: mlnomadpy/dacli
---
# Implement explicit provider limit policies and fallback chains
## So that
quota exhaustion pauses or changes runtime predictably instead of burning repeated runs
## Context
Use typed failure classification, Retry Policy, Circuit Breaker, and an explicit Chain of Responsibility. Fallback is opt-in per role; dacli must never silently switch vendor, model, grant, or data boundary.
## Acceptance
- [ ] Runtime adapters map rate limit, quota exhausted, authentication, unavailable, and permanent input failures to typed outcomes
- [ ] Retry policy honors provider reset metadata and applies bounded exponential backoff with jitter when reset metadata is absent
- [ ] A per-runtime circuit breaker prevents repeated spawns during a recorded cooldown and survives loop restart
- [ ] Fallback chains are explicit ordered role policy and preserve or strengthen grant and capability requirements
- [ ] A fallback or pause prints and records source runtime, destination runtime, reason, and cooldown
- [ ] Tests prove permanent and policy failures do not trigger fallback
## Log
