---
id: t-01KZVAT8QBZE2SWQXAK72GP5T4
kind: task
created: 2026-08-12T15:52:49Z
created_by: a-codex-loop-auditor-v4nf2e
owner: a-codex-loop-auditor-v4nf2e
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# Reject non-finite task estimates before they corrupt scheduling and timeout policy
## Acceptance
- [ ] task add --estimate Inf,Inf,Inf and task estimate <ref> --estimate Inf,Inf,Inf both fail with a usage error and leave task frontmatter unchanged
- [ ] spm.ThreePoint.Valid rejects NaN and positive or negative infinity in every estimate point, and store estimate parsing rejects non-numeric values before persisting them
- [ ] finite ordered three-point estimates continue to round-trip through task creation and resizing, and critical-path output contains neither Inf nor NaN for accepted estimates
- [ ] regression tests cover both creation and resizing command paths plus the shared ThreePoint validation
## Log
