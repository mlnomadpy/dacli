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
- 2026-08-04T20:08:05Z claimed by a-maintainer-hrnt6j
- 2026-08-04T20:29:58Z accepted by a-root
- 2026-08-04T20:29:58Z verified by `go test ./internal/features/ghmirror/` (exit 0)
- 2026-08-04T20:29:58Z completed by a-root
