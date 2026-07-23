---
id: d-131-spec-pins-the-vue-ui-to-the-existing-api-state-contract-and-forbids
kind: note
note_kind: decision
created: 2026-07-23T23:14:22Z
created_by: a-nhkth9j71n
about: [[131]]
---
# 131: spec pins the Vue UI to the existing /api/state contract and forbids inventing fields; board is count-driven, burndown is landed-per-day bars not an ideal-line
## Chose
131: spec pins the Vue UI to the existing /api/state contract and forbids inventing fields; board is count-driven, burndown is landed-per-day bars not an ideal-line
## Rejected
spec a richer UI (per-task board rows, classic descending-ideal burndown line, write actions) that the current dashboardState JSON does not carry
## Because
internal/features/dashboard/dashboard.go's dashboardState only exposes per-status counts (not task rows) and per_day landed points (not an ideal line); designing the UI around data the snapshot lacks would force either fabricated identities/lines or an unscoped server change. Pinning the spec to the real contract keeps the frontend build honest and lets the server grow the snapshot deliberately as a spec amendment.
