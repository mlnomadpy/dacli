---
id: t-01KZPWSHZCKTFF5GQG5E3CVC0E
kind: task
created: 2026-08-10T22:30:48Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Document the trust and taint model, execution boundaries, secret handling, and rollback behaviour for every mutating surface
## Acceptance
- [ ] every command declaring Mutates is covered: what it changes, what gates it, and how to undo it
- [ ] the taint model and the untrusted-content boundary are stated in one place rather than inferred from code comments
- [ ] secret handling says what dacli reads, what it never writes to a record, and where an agent token can and cannot appear
- [ ] a test asserts the doc lists every Mutates command, so the doc cannot drift as commands are added
## Log
