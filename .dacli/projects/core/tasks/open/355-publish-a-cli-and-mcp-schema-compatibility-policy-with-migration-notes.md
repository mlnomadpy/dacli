---
id: t-01KZPWSJ2P4TM80XZTQ4YZ5JYZ
kind: task
created: 2026-08-10T22:30:48Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Publish a CLI and MCP schema compatibility policy with migration notes
## Acceptance
- [ ] the policy states what is stable (exit codes, command paths, JSON shapes) and what may change without notice
- [ ] a test asserts the documented-stable JSON surfaces still parse to the documented shape, so the policy is enforced rather than asserted
- [ ] migration notes exist for any surface already changed, including the ones changed this week
## Log
