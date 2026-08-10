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
- 2026-08-10 REOPENED. This task was closed in error by the operator: `accept --force` was run against it during a batch cleanup, checking all three boxes, but NONE of the work was done. The decision issues were closed BY HAND on GitHub; `github push` still creates decision issues and still never closes them. The close was recorded as 'WITHOUT verification', which was the tool telling the truth and being ignored.
- 2026-08-10T19:16:51Z accepted by a-root
- 2026-08-10T19:16:51Z verified by `go test ./internal/features/ghmirror/` (exit 0)
- 2026-08-10T19:16:51Z deliverable: no dacli/336-github-push-creates-decision-issues-but-never-closes-them-so-records-accumulate branch — nothing to check against trunk
- 2026-08-10T19:16:51Z completed by a-root
