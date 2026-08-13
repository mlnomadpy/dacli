---
id: d-use-typed-limit-outcomes-with-retry-policy-circuit-breakers-and-explicit
kind: note
note_kind: decision
created: 2026-08-13T09:37:25Z
created_by: a-root
about: "[[406]]"
github:
  issue: 557
  repo: mlnomadpy/dacli
---
# Use typed limit outcomes with retry policy circuit breakers and explicit fallback chains
## Chose
Use typed limit outcomes with retry policy circuit breakers and explicit fallback chains
## Rejected
Substring retries and silent vendor failover
## Because
Quota exhaustion, transient rate limits, authentication failures, policy refusals, and invalid input require different actions; explicit fallback preserves operator choice and security boundaries while circuit breakers prevent repeated wasted runs.
