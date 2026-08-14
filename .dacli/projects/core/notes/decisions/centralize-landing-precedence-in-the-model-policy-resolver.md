---
id: d-centralize-landing-precedence-in-the-model-policy-resolver
kind: note
note_kind: decision
created: 2026-08-14T00:03:07Z
created_by: a-maintainer-ry3evs
about: "[[450]]"
---
# Centralize landing precedence in the model policy resolver
## Chose
Centralize landing precedence in the model policy resolver
## Rejected
Resolve project defaults independently in each landing command
## Because
A shared entity-layer resolver preserves slice isolation and gives every downstream command the same CLI-over-config-over-legacy-default result and explicit-override signal
