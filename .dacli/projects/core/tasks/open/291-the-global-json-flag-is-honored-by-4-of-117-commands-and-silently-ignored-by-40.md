---
id: t-01KZ703K8JSEZDJREGZMSG5MYA
kind: task
created: 2026-08-04T18:20:52Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 10}"
---
# The global --json flag is honored by 4 of 117 commands and silently ignored by 40 read commands
## So that
an agent can parse dacli output instead of scraping text that is formatted for humans
## Acceptance
- [ ] every read command either emits JSON under --json or refuses the flag rather than accepting and ignoring it
- [ ] an invariant test enumerates the commands and fails when a new one accepts --json without honoring it
## Log
