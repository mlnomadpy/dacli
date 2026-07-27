---
id: t-01KYHWEB2QC5251XXETY0VPW4J
kind: task
created: 2026-07-27T13:32:47Z
created_by: a-root
owner: a-root
priority: must
---
# Add end-of-options separator to all gitx argv
## So that
a --into or branch value cannot inject git flags like --upload-pack
## Acceptance
- [ ] fetch, merge, worktree, push, merge-base use -- before caller strings
## Log
