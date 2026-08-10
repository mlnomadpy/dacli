---
id: t-01KZ78MWVWPFS10MX8YNNSHCPP
kind: task
created: 2026-08-04T20:50:07Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 10}"
---
# Upgrade the loop to use the coordination the tool now has: claims, finalization, sync and preview
## So that
an unattended run gets the same protections an operator gets, instead of the loop being the one caller that skips them
## Acceptance
- [x] the build phase spawns with a claim derived from the task, and a claim conflict schedules the task into the next cycle rather than failing the wave
- [x] the land phase distinguishes a queued auto-merge from one that could not be queued
- [x] a cycle applies proposals from read-only agents before it judges whether the cycle produced anything
- [x] each cycle records a rollup of what landed, produced nothing, stalled or was blocked
## Log
- 2026-08-06T08:07:18Z claimed by a-fixer-43t0j0
- 2026-08-06T09:10:39Z accepted by a-root
- 2026-08-06T09:10:39Z closed WITHOUT verification — no --verify command was given
- 2026-08-06T09:10:39Z completed by a-root
- 2026-08-08T11:07:20Z a-fixer-43t0j0: PR opened: https://github.com/mlnomadpy/dacli/pull/387 (event 01KZB2RS393PKQ9R8WDW8MXHQH)
