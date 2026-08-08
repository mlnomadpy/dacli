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
- [x] every read command either emits JSON under --json or refuses the flag rather than accepting and ignoring it
- [x] an invariant test enumerates the commands and fails when a new one accepts --json without honoring it
## Log
- 2026-08-05T13:35:30Z claimed by a-fixer-fqabnj
- 2026-08-05T14:01:17Z accepted by a-root
- 2026-08-05T14:01:17Z closed WITHOUT verification — no --verify command was given
- 2026-08-05T14:01:17Z completed by a-root
