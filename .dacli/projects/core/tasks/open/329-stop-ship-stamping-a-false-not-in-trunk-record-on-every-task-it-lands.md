---
id: t-01KZP94HFA9YJZXMVAJ0KCD47H
kind: task
created: 2026-08-10T16:47:16Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 6}"
---
# Stop ship stamping a false 'NOT in trunk' record on every task it lands
## Acceptance
- [ ] after a successful ship, no task closed by that run carries a Log line asserting its branch is NOT in trunk
- [ ] the landing verdict is still recorded truthfully when ship's integrate step fails, so a genuinely unlanded close is still visible
- [ ] the fix addresses the durable Log line written at acceptance.go:267, not only the stderr warning that --allow-unlanded silences
- [ ] a test drives the accept-then-integrate order ship uses and asserts the final task record matches where the work actually ended up
## Log
- 2026-08-10T16:53:26Z claimed by a-fixer-g7xspa
