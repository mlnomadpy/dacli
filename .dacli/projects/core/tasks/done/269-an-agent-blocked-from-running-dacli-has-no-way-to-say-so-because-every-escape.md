---
id: t-01KZ6RTFZEG3HQDB2ZEXV7MBE4
kind: task
created: 2026-08-04T16:13:34Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# An agent blocked from running dacli has no way to say so, because every escape hatch is dacli
## So that
the one failure that silences every other report is still reportable
## Acceptance
- [x] a child can record 'I am blocked and why' using only a plain file write, with no dacli invocation
- [x] agents and wait surface that report as a distinct state, not as a normal completion
- [x] the child prompt tells the agent this channel exists and when to use it
## Log
- 2026-08-05T13:59:42Z claimed by a-junior-rxxbd6
- 2026-08-06T08:03:57Z accepted by a-root
- 2026-08-06T08:03:57Z verified by `go test ./internal/features/execution/...` (exit 0)
- 2026-08-06T08:03:57Z completed by a-root
- 2026-08-08T11:07:20Z a-junior-rxxbd6: PR opened: https://github.com/mlnomadpy/dacli/pull/377 (event 01KZ93N3DGQ351WCWWCH04A6FN)
- 2026-08-08T11:07:20Z status done proposed by a-junior-rxxbd6, applied (event 01KZ93N6VZX8N5TAAQSXTYP57K)
