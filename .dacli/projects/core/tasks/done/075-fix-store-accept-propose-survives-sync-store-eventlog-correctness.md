---
id: t-01KY5JSH65AYYPHHGJ9Z5QR6EA
kind: task
created: 2026-07-22T18:53:14Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# FIX-store: accept-propose survives sync; store/eventlog correctness
## Acceptance
- [x] an agent's accept-propose event is NOT silently dropped by eventlog.Sync — it survives so dacli accept can apply it (Sync skips accept-propose comments, or accept consumes them before Sync)
- [x] any remaining store/eventlog correctness findings from AUDIT2 R5 are addressed (verify against the filed findings)
- [x] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T18:53:33Z claimed by a-3mjddwtxf4
- 2026-07-22T19:31:56Z accepted by a-root
- 2026-07-22T19:31:56Z completed by a-root
