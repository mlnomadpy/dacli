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
- [x] spawn refuses or warns when os.Executable does not match a path the runtime's allowlist permits
- [x] a test covers the mismatch, using the real cc-rw shape (an absolute-path Bash rule)
## Log
- 2026-08-04T14:36:23Z claimed by a-maintainer-c76h39
- 2026-08-04T16:05:16Z accepted by a-root
- 2026-08-04T16:05:16Z verified by `go test ./internal/store/ ./internal/features/execution/` (exit 0)
- 2026-08-04T16:05:16Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/334 (event 01KZ6RC1HEKWC2Y3HCY0B7C787)
