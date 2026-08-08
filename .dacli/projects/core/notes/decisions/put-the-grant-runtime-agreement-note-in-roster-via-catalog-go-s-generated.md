---
id: d-put-the-grant-runtime-agreement-note-in-roster-via-catalog-go-s-generated
kind: note
note_kind: decision
created: 2026-08-04T11:14:54Z
created_by: a-maintainer-vgqd2d
about: "[[253]]"
---
# put the grant/runtime-agreement note in ROSTER via catalog.go's generated preamble, not a hand-edit of the generated file
## Chose
put the grant/runtime-agreement note in ROSTER via catalog.go's generated preamble, not a hand-edit of the generated file
## Rejected
hand-editing docs/ROSTER.md directly
## Because
ROSTER.md is a one-way generated view (renderCatalog in internal/features/catalog/catalog.go); a hand-edit is wiped on the next dacli catalog. Adding the line to the deterministic preamble constant keeps it persistent, and the committed ROSTER.md is edited to match verbatim.
