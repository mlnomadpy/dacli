---
id: t-01KZP8JE4GS64222HJ5JBPAAN8
kind: task
created: 2026-08-10T16:37:23Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# Audit the kind-verb table against the verbs real tasks actually use, so review work stops routing to implementers
## Acceptance
- [x] the verbs that misrouted are named with the task titles that produced them (324 'Falsify', 325 'Trace' both routed to fixer)
- [x] high-signal review/research verbs are added, and each addition is justified by a title a human or agent would plausibly write
- [x] a test asserts each newly-added verb routes to its kind, and that an unmatched verb still falls through to the phase then implementer
- [x] the map stays deliberately small: no verb is added that could plausibly lead an implementation task
## Log
- 2026-08-10T16:53:18Z accepted by a-root
- 2026-08-10T16:53:18Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T16:53:18Z deliverable: no dacli/326-audit-the-kind-verb-table-against-the-verbs-real-tasks-actually-use-so-review branch — nothing to check against trunk
- 2026-08-10T16:53:18Z completed by a-root
