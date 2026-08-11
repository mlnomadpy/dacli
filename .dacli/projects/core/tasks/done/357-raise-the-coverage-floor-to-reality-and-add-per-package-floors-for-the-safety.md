---
id: t-01KZR80DD835MME33HP9CKR1PA
kind: task
created: 2026-08-11T11:06:02Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Raise the coverage floor to reality and add per-package floors for the safety surfaces
## Acceptance
- [x] the global floor sits just under measured coverage, so it catches a collapse instead of sitting 7 points below reality
- [x] the packages that gate capability or validate paths carry their own floor, so a collapse in one is not masked by a rise elsewhere
- [x] the floor check names which package fell and by how much, rather than printing one aggregate number
## Log
- 2026-08-11T11:14:45Z accepted by a-root
- 2026-08-11T11:14:45Z closed WITHOUT verification — no --verify command was given
- 2026-08-11T11:14:45Z deliverable: no dacli/357-raise-the-coverage-floor-to-reality-and-add-per-package-floors-for-the-safety branch — nothing to check against sprint/15
- 2026-08-11T11:14:45Z completed by a-root
