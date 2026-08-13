---
id: d-use-a-provider-neutral-modelprofile-on-roles-with-legacy-field-migration
kind: note
note_kind: decision
created: 2026-08-13T09:46:25Z
created_by: a-codex-maintainer-b3scj1
about: "[[403]]"
---
# Use a provider-neutral ModelProfile on roles with legacy field migration
## Chose
Use a provider-neutral ModelProfile on roles with legacy field migration
## Rejected
Keep model-name substring tiers or add provider-specific catalog fields
## Because
The routing engine needs stable cost, capacity, context, and capability metadata independent of runtime adapters, while existing role files must continue to route unchanged.
