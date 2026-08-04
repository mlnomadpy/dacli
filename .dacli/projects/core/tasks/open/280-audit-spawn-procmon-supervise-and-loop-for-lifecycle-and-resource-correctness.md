---
id: t-01KZ6SBB2971QQGMXS8ZTC55VH
kind: task
created: 2026-08-04T16:22:46Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 10}"
---
# Audit spawn, procmon, supervise and loop for lifecycle and resource correctness
## So that
a long unattended run does not accumulate zombies, leak budget, or lose track of a child it started
## Acceptance
- [ ] every path that starts a process is checked for who reaps it and what happens when the parent dies first
- [ ] timeout, kill and detach interactions are traced end to end, with the surviving-state named for each
- [ ] token and budget accounting is checked for double counting and for spend that escapes the window entirely
## Log
- 2026-08-04T16:25:07Z claimed by a-go-auditor-a2hqh6
- 2026-08-04T18:18:12Z finding by a-go-auditor-a2hqh6: runStillLive's GroupAlive fallback has no PID-identity check — a recycled pgid resurrects a finished run as live (event 01KZ6SVY707GQEAHERG0HCAW75)
