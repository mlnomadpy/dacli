---
id: t-01KYHWEWKSDDBF2C3FF0FT32GJ
kind: task
created: 2026-07-27T13:33:05Z
created_by: a-root
owner: a-root
priority: must
---
# Front.Set corrupts task files on newline or trailing hash
## So that
a free-text flag value cannot make a task permanently invisible
## Acceptance
- [ ] Set quotes or rejects newline and control chars
- [ ] value with space-hash round-trips through Get
## Log
