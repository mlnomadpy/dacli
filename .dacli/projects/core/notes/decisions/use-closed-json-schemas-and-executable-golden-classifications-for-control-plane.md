---
id: d-use-closed-json-schemas-and-executable-golden-classifications-for-control-plane
kind: note
note_kind: decision
created: 2026-08-13T14:24:22Z
created_by: a-fixer-6nxvsp
about: "[[408]]"
---
# Use closed JSON Schemas and executable golden classifications for control-plane v1
## Chose
Use closed JSON Schemas and executable golden classifications for control-plane v1
## Rejected
Document the protocol only in prose or add runtime SaaS integration code
## Because
Closed schemas make the metadata allowlist machine-checkable while fixture tests lock deterministic Inbox outcomes without making a remote service authoritative
