---
id: t-01KZB5EJNXTDN2VFSGTPH4KZRJ
kind: task
created: 2026-08-06T09:11:12Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 0.5, probable: 1, pessimistic: 3}"
---
# land 287's orphaned ro/rw-lifecycle preamble test — it fails on main, so template and test drifted
## So that
the ro preamble's no-commit guarantee is enforced by a test instead of prose
## Acceptance
- [x] the RO preamble contains no commit/PR instructions and the test asserting it passes on trunk
- [x] the RW preamble still describes claim-work-commit-pr and the test asserts the difference
## Log
- 2026-08-08T12:13:05Z accepted by a-root
- 2026-08-08T12:13:05Z verified by `go build ./...` (exit 0)
- 2026-08-08T12:13:05Z completed by a-root
