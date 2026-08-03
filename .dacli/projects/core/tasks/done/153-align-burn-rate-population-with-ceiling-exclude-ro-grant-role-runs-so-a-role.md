---
id: t-01KYFP722KW8GKJM905SX3FB4M
kind: task
created: 2026-07-26T17:05:28Z
created_by: a-1hwz5pcjva
owner: a-root
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
- [x] internal/features/dashboard/burn.go reviewerRoles excludes a run whose role has grant==model.GrantRO (not only role_kind=='reviewer'), so the Rate counts the same population the Ceiling is calibrated over
- [x] a unit test in internal/features/dashboard proves a run from an ro-grant role that omits role_kind is dropped from burnSeries' per-run Rate, and an rw implementer run is still counted
- [x] .dacli/roles/reviewer.md is backfilled with role_kind: reviewer as belt-and-suspenders data hygiene
- [x] the burn.go doc comment on reviewerRoles is updated to state the ro-grant exclusion rule and why (an ro role never becomes a calibration sample)
- [x] go build ./... clean and go test ./internal/... green
## Log
- 2026-07-26T21:22:16Z claimed by a-sz4h77f3rf
- 2026-07-26T21:24:53Z adopted by a-root (owner a-1hwz5pcjva orphaned)
- 2026-07-26T21:24:53Z accepted by a-root
- 2026-07-26T21:24:53Z completed by a-root
- 2026-08-03T22:38:15Z a-sz4h77f3rf: PR opened: https://github.com/mlnomadpy/dacli/pull/265 (event 01KYG51TY163KMEKAV4W9SJF38)
