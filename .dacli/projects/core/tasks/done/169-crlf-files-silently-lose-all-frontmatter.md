---
id: t-01KYHWEWK7BXBP6JJQKWC8WK77
kind: task
created: 2026-07-27T13:33:04Z
created_by: a-root
owner: a-root
priority: must
---
# CRLF files silently lose all frontmatter
## So that
dacli works on Windows clones where autocrlf rewrites line endings
## Acceptance
- [x] mdstore parses --- headers with CRLF
- [x] a gitattributes pins workspace md to LF
## Log
- 2026-07-27T23:03:03Z completed by a-root
