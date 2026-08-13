---
id: d-keep-roster-sandbox-guidance-in-the-catalog-generator
kind: note
note_kind: decision
created: 2026-08-13T13:33:03Z
created_by: a-fixer-xdgjvd
about: "[[414]]"
github:
  issue: 577
  repo: mlnomadpy/dacli
---
# Keep roster sandbox guidance in the catalog generator
## Chose
Keep roster sandbox guidance in the catalog generator
## Rejected
Hand-edit docs/ROSTER.md after generation
## Because
ROSTER.md is a generated projection; correcting renderCatalog makes every regeneration durable and preserves provenance.
