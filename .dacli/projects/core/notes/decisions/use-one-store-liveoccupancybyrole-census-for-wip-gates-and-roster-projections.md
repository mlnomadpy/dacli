---
id: d-use-one-store-liveoccupancybyrole-census-for-wip-gates-and-roster-projections
kind: note
note_kind: decision
created: 2026-08-22T15:27:52Z
created_by: a-maintainer-qt88ce
about: "[[t-01M0AK4XK4M7CTJ6DXRKFW8XWG]]"
---
# Use one store.LiveOccupancyByRole census for WIP gates and roster projections
## Chose
Use one store.LiveOccupancyByRole census for WIP gates and roster projections
## Rejected
Continue adapting the minted/live/finished holdsWIPSlot predicate or duplicate liveness scans in team and dashboard
## Because
Current capacity is process truth, while role-removal provenance must remain fail-closed for never-started identities; a store-level census keeps feature slices consistent without sideways imports.
