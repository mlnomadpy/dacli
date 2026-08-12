---
id: t-01KZVK9A70V2YPA2Y6DJZ50V2D
kind: task
created: 2026-08-12T18:20:51Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 507
  repo: mlnomadpy/dacli
---
# Upgrade Codex loop roles with dacli and GitHub-first operating knowledge
## So that
the restarted loop can coordinate, verify, recover, and publish work without rediscovering operator-only workflow rules
## Acceptance
- [x] codex-maintainer and codex-loop-auditor both select the using-dacli workspace skill and have their versions bumped
- [x] Both role methods require GitHub-backed task synchronization, dry-run before public mutation, and small idempotent recovery batches after partial remote application
- [x] The maintainer method requires dacli wait/sync/commit and independent acceptance evidence; the auditor method requires duplicate checks across local tasks and linked GitHub issues
- [x] dacli skill compile --dry-run and dacli preflight succeed for both roles on codex-rw
## Log
- 2026-08-12T18:21:34Z completed by a-root
