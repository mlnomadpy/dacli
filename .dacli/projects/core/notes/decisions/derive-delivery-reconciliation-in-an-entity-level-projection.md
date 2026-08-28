---
id: d-derive-delivery-reconciliation-in-an-entity-level-projection
kind: note
note_kind: decision
created: 2026-08-28T12:53:32Z
created_by: a-maintainer-k0gy77
about: "[[t-01M146BA62817V08T9P6D6REKT]]"
---
# Derive delivery reconciliation in an entity-level projection
## Chose
Derive delivery reconciliation in an entity-level projection
## Rejected
Implement reconciliation inside the insight feature or persist a delivery ledger
## Because
doctor and future recovery consumers need identical classifications without feature imports, while derived state must remain read-only
