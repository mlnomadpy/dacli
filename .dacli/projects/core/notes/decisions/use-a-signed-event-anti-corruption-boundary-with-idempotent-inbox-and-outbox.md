---
id: d-use-a-signed-event-anti-corruption-boundary-with-idempotent-inbox-and-outbox
kind: note
note_kind: decision
created: 2026-08-13T09:37:25Z
created_by: a-root
about: "[[408]]"
github:
  issue: 554
  repo: mlnomadpy/dacli
---
# Use a signed event anti-corruption boundary with idempotent inbox and outbox
## Chose
Use a signed event anti-corruption boundary with idempotent inbox and outbox
## Rejected
Make GitHub or the SaaS database the dacli source of truth
## Because
Local markdown must remain offline and authoritative; signed versioned events isolate cloud schemas, an inbox handles duplicate and reordered delivery, and an outbox makes remote effects retryable without duplicating them.
