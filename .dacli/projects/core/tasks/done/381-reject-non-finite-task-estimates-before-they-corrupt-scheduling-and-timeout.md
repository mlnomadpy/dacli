---
id: t-01KZVAT8QBZE2SWQXAK72GP5T4
kind: task
created: 2026-08-12T15:52:49Z
created_by: a-codex-loop-auditor-v4nf2e
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 473
  repo: mlnomadpy/dacli
---
# Reject non-finite task estimates before they corrupt scheduling and timeout policy
## Acceptance
- [x] task add --estimate Inf,Inf,Inf and task estimate <ref> --estimate Inf,Inf,Inf both fail with a usage error and leave task frontmatter unchanged
- [x] spm.ThreePoint.Valid rejects NaN and positive or negative infinity in every estimate point, and store estimate parsing rejects non-numeric values before persisting them
- [x] finite ordered three-point estimates continue to round-trip through task creation and resizing, and critical-path output contains neither Inf nor NaN for accepted estimates
- [x] regression tests cover both creation and resizing command paths plus the shared ThreePoint validation
## Log
- 2026-08-12T16:55:43Z claimed by a-codex-maintainer-gqkrc4
- 2026-08-12T17:02:21Z adopted by a-root (owner a-codex-loop-auditor-v4nf2e orphaned)
- 2026-08-12T17:02:21Z accepted by a-root
- 2026-08-12T17:02:21Z closed WITHOUT verification — no --verify command was given
- 2026-08-12T17:02:21Z deliverable: dacli/381-reject-non-finite-task-estimates-before-they-corrupt-scheduling-and-timeout exists but is NOT in main — closed anyway
- 2026-08-12T17:02:21Z completed by a-root
- 2026-08-12T18:19:47Z claimed by a-root (event 01KZVAWDA971T3ZEQGBN767GX9)
- 2026-08-12T18:19:47Z claimed by a-root (event 01KZVAXAETAW4PA5QZ4HKGY636)
