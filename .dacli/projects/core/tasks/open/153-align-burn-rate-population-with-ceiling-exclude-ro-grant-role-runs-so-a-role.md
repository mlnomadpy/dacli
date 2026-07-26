---
id: t-01KYFP722KW8GKJM905SX3FB4M
kind: task
created: 2026-07-26T17:05:28Z
created_by: a-1hwz5pcjva
owner: a-1hwz5pcjva
priority: should
estimate: {optimistic: 1, probable: 2, pessimistic: 4}
github:
  issue: 249
  repo: mlnomadpy/dacli
---
# Align burn Rate population with Ceiling: exclude ro-grant role runs so a role_kind-less reviewer no longer dilutes the 1.5x yell
## So that
the burn overspend alert task 149 shipped cannot be silently defeated by any read-only role (reviewer.md) whose file omits role_kind
## Acceptance
- [ ] internal/features/dashboard/burn.go reviewerRoles excludes a run whose role has grant==model.GrantRO (not only role_kind=='reviewer'), so the Rate counts the same population the Ceiling is calibrated over
- [ ] a unit test in internal/features/dashboard proves a run from an ro-grant role that omits role_kind is dropped from burnSeries' per-run Rate, and an rw implementer run is still counted
- [ ] .dacli/roles/reviewer.md is backfilled with role_kind: reviewer as belt-and-suspenders data hygiene
- [ ] the burn.go doc comment on reviewerRoles is updated to state the ro-grant exclusion rule and why (an ro role never becomes a calibration sample)
- [ ] go build ./... clean and go test ./internal/... green
## Log
