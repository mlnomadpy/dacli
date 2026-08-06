---
id: t-01KZB4SAYXVFY0EXR9C40JG58R
kind: task
created: 2026-08-06T08:59:36Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# github link is not auto-detected from an existing origin, and its absence silently defers accept forever
## So that
PR-merge confirmation works out of the box when origin and gh auth exist, and says what is missing when it cannot
## Acceptance
- [ ] a workspace with a gh-authed origin and no linked repo either auto-links on first use or names the missing github link in the deferred-accept message
- [ ] the loop's deferred-accept line states the reason (no link, no PR found, PR open) rather than one unconditional sentence
## Log
