---
id: t-01KZRFRHF2KWYB8H639SDG6JCS
kind: task
created: 2026-08-11T13:21:32Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# MCP argument coercion silently returns a zero value for an uninterpretable argument
## Acceptance
- [ ] an argument present but uninterpretable is refused by name rather than coerced to the zero value
- [ ] the refusal reaches the MCP client as an error, so an agent learns its call was malformed instead of getting a wrong default
- [ ] a test asserts a nonsense dry_run value is refused rather than silently read as false
## Log
