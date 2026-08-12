---
id: d-promise-power-loss-durability-for-workspace-records
kind: note
note_kind: decision
created: 2026-08-12T19:23:56Z
created_by: a-codex-maintainer-zf35yj
about: "[[368]]"
github:
  issue: 529
  repo: mlnomadpy/dacli
---
# Promise power-loss durability for workspace records
## Chose
Promise power-loss durability for workspace records
## Rejected
Document atomic rename as process-crash safety only
## Because
dacli presents tasks and events as durable shared state; returning success before syncing file data and the renamed directory would let acknowledged agent work disappear after power loss.
