---
id: f-live-roster-occupancy-had-reused-conservative-minted-identity-provenance
kind: note
note_kind: finding
created: 2026-08-22T15:27:52Z
created_by: a-maintainer-qt88ce
about: "[[t-01M0AK4XK4M7CTJ6DXRKFW8XWG]]"
severity: major
---
# Live roster occupancy had reused conservative minted-identity provenance
internal/store/roles.go ActiveInRole used holdsWIPSlot, whose deliberate issue #690 behavior counts minted-but-never-run identities; internal/features/teamops/teamops.go and internal/features/dashboard/roster.go projected that as active WIP despite no live proc. The new LiveOccupancyByRole instead counts only procmon.AliveRecord-backed runs while RemoveRole retains holdsWIPSlot.
