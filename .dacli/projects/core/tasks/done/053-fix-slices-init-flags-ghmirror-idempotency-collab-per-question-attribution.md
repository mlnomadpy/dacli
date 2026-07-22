---
id: t-01KY59YW1HCJ8NK318MTVPF0YM
kind: task
created: 2026-07-22T16:18:52Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 4, pessimistic: 6}
---
# FIX slices: init flags, ghmirror idempotency, collab per-question attribution, selfreport timeout
## Acceptance
- [x] dacli init honors --template and --roster (its Brief-advertised, spec-documented flags) instead of silently ignoring them
- [x] collab threads attribute answers per-QUESTION not per-task; selfreport gh subprocesses use a context timeout so a hung gh cannot block dacli/mcp serve
- [x] ghmirror marker-idempotency hardened against duplicate on eventually-consistent GitHub search; governance stale docstring corrected
- [x] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T16:19:10Z claimed by a-sfa41hsara
- 2026-07-22T16:40:39Z accepted by a-root
- 2026-07-22T16:40:39Z completed by a-root
