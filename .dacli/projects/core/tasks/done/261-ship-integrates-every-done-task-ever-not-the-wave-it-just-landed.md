---
id: t-01KZ6AZZGPF88H6NYZ1MGW303W
kind: task
created: 2026-08-04T12:11:53Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# ship integrates every done task ever, not the wave it just landed
## So that
ship stays usable on a project with a long history instead of getting more dangerous the longer it runs
## Acceptance
- [x] ship integrates only tasks closed by this run, or an explicit window, not the full done set
- [x] a regression test covers a workspace whose done set is much larger than the wave
## Log
- 2026-08-04T12:19:52Z claimed by a-maintainer-0b7kdr
- 2026-08-04T12:52:22Z accepted by a-root
- 2026-08-04T12:52:22Z verified by `go test ./internal/features/ship/` (exit 0)
- 2026-08-04T12:52:22Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/327 (event 01KZ6DAWHPPHR7SGHZPFGDKXB3)
