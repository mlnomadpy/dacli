---
id: t-01KZ6SAHPQ9ZB2XNTMWC3HPCV5
kind: task
created: 2026-08-04T16:22:20Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 10}"
---
# Audit the 117-command surface for coherence: naming, flags, JSON coverage, missing inverses
## So that
an agent can predict a command it has not used from the ones it has
## Acceptance
- [x] flags that mean the same thing under different names, or the same name meaning different things, are listed with both call sites
- [x] commands that mutate but offer no --dry-run, and read commands with no --json, are listed
- [x] operations with no inverse (something creatable but not removable) are named
## Log
- 2026-08-04T16:24:48Z claimed by a-go-auditor-2ednq4
- 2026-08-04T18:18:12Z finding by a-go-auditor-2ednq4: global --json flag is honored by only 4 of ~117 commands; ~40 read commands silently ignore it (exit 0, human text) (event 01KZ6SQ3XJCRC5NBH92RXAM0N6)
- 2026-08-04T18:18:12Z finding by a-go-auditor-2ednq4: one token-ceiling concept lives under 4 flag names, and --budget/--all are homonyms across commands (event 01KZ6SQG9E120445RZQW9B7ZV4)
- 2026-08-04T18:18:12Z finding by a-go-auditor-2ednq4: ten object types can be created but have no removal/inverse command; only project/worktree/agent do (event 01KZ6SQXX25AWBMXP4KTC2X0M4)
- 2026-08-04T18:18:12Z finding by a-go-auditor-2ednq4: every github remote-mutating command lacks --dry-run, while local integrate/ship/loop/worktree-prune offer one (event 01KZ6SR9HXKSCD16T8XX9AAXNB)
- 2026-08-04T18:26:18Z accepted by a-root
- 2026-08-04T18:26:18Z verified by `ls .dacli/projects/core/notes/findings/ | wc -l | awk '{exit ($1<16)}'` (exit 0)
- 2026-08-04T18:26:18Z completed by a-root
