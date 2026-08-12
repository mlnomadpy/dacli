---
id: t-01KZVAWWJV12EC8VW19FSCBY7Q
kind: task
created: 2026-08-12T15:54:15Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 477
  repo: mlnomadpy/dacli
---
# Fix status probes that finalize live runs when process visibility is restricted
## Acceptance
- [ ] agents --tail and runs list do not rewrite a running run to no-visible-result solely because the current caller cannot observe its PID or process group
- [ ] a run with a live guardian and runtime remains running when sampled from a restricted process-visibility context, and later wait finalizes the real outcome
- [ ] status reads are non-mutating; only wait, watchdog, explicit kill, or an identity-authenticated lifecycle reconciler may finalize outcome.md
- [ ] regression tests reproduce a false-negative liveness probe while the recorded guardian remains alive and prove the run outcome is not downgraded
## Log
- 2026-08-12T18:24:00Z claimed by a-codex-maintainer-j8jbvt
