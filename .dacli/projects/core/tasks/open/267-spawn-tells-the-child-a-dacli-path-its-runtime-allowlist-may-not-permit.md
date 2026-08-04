---
id: t-01KZ6HET45TV024STVGQX3QA5B
kind: task
created: 2026-08-04T14:04:51Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# spawn tells the child a dacli path its runtime allowlist may not permit
## So that
a child can always run the binary the preamble tells it to run
## Acceptance
- [ ] spawn refuses or warns when os.Executable does not match a path the runtime's allowlist permits
- [ ] a test covers the mismatch, using the real cc-rw shape (an absolute-path Bash rule)
## Log
- 2026-08-04T14:36:23Z claimed by a-maintainer-c76h39
