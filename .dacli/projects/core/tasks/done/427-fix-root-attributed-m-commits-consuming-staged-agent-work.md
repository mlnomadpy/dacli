---
id: t-01KZXYN54AETKFJ0D7SH406587
kind: task
created: 2026-08-13T16:18:02Z
created_by: a-codex-loop-auditor-et4f9e
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 596
  repo: mlnomadpy/dacli
---
# Fix root-attributed -m commits consuming staged agent work
## So that
a worker's verified diff cannot be silently committed under the owner identity and leave the documented worker commit reporting a misleading no-op
## Acceptance
- [x] A deterministic regression reproduces the task 422/423 sequence: an rw child stages claimed files in its isolated worktree, the malformed/root commit path runs, and the test observes that current behavior creates an a-root commit titled -m before the child's documented commit reports nothing staged
- [x] The commit/lifecycle path prevents any owner or malformed invocation from consuming an rw child's staged work; the resulting implementation commit carries that child's Dacli-Agent, Dacli-Role, and Dacli-Task trailers and the requested subject
- [x] Failure and concurrency paths report the actor and preserve recoverable staged work without creating or pushing a malformed commit; focused tests and go test ./... pass
## Log
- 2026-08-13T19:02:15Z claimed by a-codex-maintainer-mjejj8
- 2026-08-13T19:18:19Z adopted by a-root (owner a-codex-loop-auditor-et4f9e orphaned)
- 2026-08-13T19:18:19Z accepted by a-root
- 2026-08-13T19:18:19Z verified by `GOCACHE=/private/tmp/dacli-427-main-gocache go test ./internal/features/vcs` (exit 0) in branch main at 0200b11 — proves that tree builds, not that the work is in trunk
- 2026-08-13T19:18:19Z deliverable: dacli/427-fix-root-attributed-m-commits-consuming-staged-agent-work is merged into main
- 2026-08-13T19:18:19Z completed by a-root
- 2026-08-13T19:25:42Z claimed by a-root (event 01KZXYYX4ZKJYHHRKT4DSZ3AZ6)
- 2026-08-13T19:25:42Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/598 (event 01KZY8JRTG9GHGRNEAQ28C2DW1)
