---
id: t-01KZ6SAYED1SGWXJE4ZZQNKMCK
kind: task
created: 2026-08-04T16:22:33Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 10}"
---
# Audit core plumbing for swallowed errors, partial writes and non-atomic updates
## So that
a crash or a concurrent writer cannot leave the workspace in a state no command can read
## Acceptance
- [ ] every place an error is discarded in store, eventlog, mdstore, workspace and gitx is either justified in a comment or filed
- [ ] each finding names the concrete interleaving or failure point that corrupts state, not just the discarded error
- [ ] read-modify-write sequences without a lock or an atomic rename are listed
## Log
- 2026-08-04T16:24:59Z claimed by a-go-auditor-s12cpg
- 2026-08-04T18:18:12Z finding by a-go-auditor-s12cpg: propose:done sync closes a task via MoveTask, bypassing CloseTask's completed-by stamp and acceptance verification (event 01KZ6SNV79GSAS8PEYB48W14JR)
- 2026-08-04T18:18:12Z finding by a-go-auditor-s12cpg: 279 audit coverage: discarded errors in core plumbing are justified-in-comment; read-modify-write sequences enumerated (event 01KZ6SQNZNE7WHE7Y9SDEB8RHD)
- 2026-08-04T18:18:12Z status done proposed by a-go-auditor-s12cpg, applied (event 01KZ6SR16441HT7TZ1NV61SV21)
