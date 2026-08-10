---
id: t-01KZ704C30GYJJ045MC2KKCZ3V
kind: task
created: 2026-08-04T18:21:17Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 3, pessimistic: 6}"
---
# Every GitHub remote-mutating command lacks --dry-run while the local ones have it
## So that
the commands that write to a public repository are the ones you can preview, not the only ones you cannot
## Acceptance
- [x] push, pull, sync, project, release and codeowners accept --dry-run and print exactly what they would create, adopt or close
- [x] the preview is derived from the same code path as the real run rather than a parallel description of it
## Log
- 2026-08-05T13:03:23Z accepted by a-root
- 2026-08-05T13:03:23Z verified by `go build ./...` (exit 0)
- 2026-08-05T13:03:23Z completed by a-root
- 2026-08-08T11:07:20Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/368 (event 01KZ77H4KV9JVMVT37K882CVR7)
