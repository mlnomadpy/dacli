---
id: d-keep-legacy-price-migration-at-the-role-persistence-boundary
kind: note
note_kind: decision
created: 2026-08-13T09:59:22Z
created_by: a-codex-maintainer-vrytxy
about: "[[403]]"
---
# Keep legacy price migration at the role persistence boundary
## Chose
Keep legacy price migration at the role persistence boundary
## Rejected
Restore vendor-name cost heuristics inside internal/team or leave old roles unpriced
## Because
The store is the anti-corruption boundary for old role files; internal/team remains a pure provider-neutral declared-tier policy while existing rosters retain their historical ordering.
