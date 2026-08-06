---
id: t-01KZB4SQRHF2P7TME8RBR2BT5J
kind: task
created: 2026-08-06T08:59:49Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 3, pessimistic: 6}"
---
# an autonomous agent created a GitHub repo and pushed without operator consent
## So that
outward, hard-to-reverse side effects (repo creation, first push to a new origin) happen only when the operator explicitly allowed them
## Acceptance
- [ ] gh repo create and setting a new origin from inside a spawned agent's PR flow are refused unless an explicit operator flag or config allows them
- [ ] the refusal names the flag that grants consent, and the event log records who attempted the outward write
## Log
