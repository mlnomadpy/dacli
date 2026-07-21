---
id: t-01KY3CGC860G5DSQ6SF12GHFXW
kind: task
created: 2026-07-21T22:24:54Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 3, pessimistic: 6}
---
# dacli report: file dacli-tool bugs upstream via gh
## So that
problems agents hit in dacli itself flow back to the tool's issue tracker
## Acceptance
- [x] report files a gh issue to the configured dacli repo with version and OS context
- [x] default repo is the tool's own, overridable by env; refuses cleanly if gh is absent or unauth
- [x] it is an explicit command, never automatic — no surprise outbound issues
## Log
- 2026-07-21T22:30:43Z claimed by a-root
- 2026-07-21T22:35:13Z completed by a-root
