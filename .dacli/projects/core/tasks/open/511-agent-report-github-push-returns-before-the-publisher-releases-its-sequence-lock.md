---
id: t-01M1068MTFPQ6YFVQG204M2EX4
kind: task
created: 2026-08-26T23:25:11Z
created_by: a-root
owner: a-root
github:
  issue: 797
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] github push returns before the publisher releases its sequence lock
## Context
Adopted from GitHub issue #797.

A multi-task github push printed only its plan and the invoking process returned successfully, while a child process continued creating issues for over a minute and retained the github-push sequence lock. A second idempotent push then failed because the lock was still held. The command should remain attached until publication completes, stream progress, and return only after releasing the lock.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] A deterministic multi-task publisher fixture blocks after planning and proves the public `github push` process remains attached while publication is active.
- [ ] Progress is streamed through the invoking command and its exit status reflects the terminal publisher result.
- [ ] The sequence lock is released before the command returns on success, failure, cancellation, and timeout.
- [ ] An immediate idempotent second push never observes a live child or lock left behind by the completed first invocation.
- [ ] Process-tree and lock-ownership regressions run under the race detector without depending on GitHub network access.
- [ ] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-28T00:36:13Z claimed by a-fixer-0mrzjc
