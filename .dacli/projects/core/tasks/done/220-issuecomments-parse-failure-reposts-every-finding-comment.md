---
id: t-01KZ4WDDGSJHF9EB6CZXBQY24X
kind: task
created: 2026-08-03T22:37:51Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 0.5, probable: 1, pessimistic: 2.5}"
---
# issueComments parse failure reposts every finding comment
## So that
a transient parse error does not duplicate finding comments
## Acceptance
- [x] an unparseable comment list is a failure not an empty list
## Log
- 2026-08-04T12:03:06Z claimed by a-maintainer-bqv5pa
- 2026-08-04T12:11:42Z accepted by a-root
- 2026-08-04T12:11:42Z verified by `go test ./internal/features/ghmirror/` (exit 0)
- 2026-08-04T12:11:42Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/318 (event 01KZ6B0HDVRRNWH53H37BY8VPF)
