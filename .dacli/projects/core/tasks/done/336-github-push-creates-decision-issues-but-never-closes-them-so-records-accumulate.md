---
id: t-01KZPDWTRP1SF1BPWB50022Y3F
kind: task
created: 2026-08-10T18:10:27Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# github push creates decision issues but never closes them, so records accumulate as open work forever
## Acceptance
- [x] a decision note mirrored as an issue is either filed closed, or closed on the next push, so the issue list holds work rather than records
- [x] the behaviour is chosen deliberately and stated in the command's brief, not left implicit
- [x] a test asserts a decision issue does not remain open after a second push
## Log
- 2026-08-10T18:49:58Z accepted by a-root
- 2026-08-10T18:49:58Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T18:49:58Z deliverable: no dacli/336-github-push-creates-decision-issues-but-never-closes-them-so-records-accumulate branch — nothing to check against trunk
- 2026-08-10T18:49:58Z completed by a-root
