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
- [x] agents --tail and runs list do not rewrite a running run to no-visible-result solely because the current caller cannot observe its PID or process group
- [x] a run with a live guardian and runtime remains running when sampled from a restricted process-visibility context, and later wait finalizes the real outcome
- [x] status reads are non-mutating; only wait, watchdog, explicit kill, or an identity-authenticated lifecycle reconciler may finalize outcome.md
- [x] regression tests reproduce a false-negative liveness probe while the recorded guardian remains alive and prove the run outcome is not downgraded
## Log
- 2026-08-12T18:24:00Z claimed by a-codex-maintainer-j8jbvt
- 2026-08-12T18:56:43Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/509 (event 01KZVKXRG40ZPQ054RYA4RGG1X)
- 2026-08-12T19:00:43Z accepted by a-root
- 2026-08-12T19:00:43Z verified by `GOCACHE=/private/tmp/dacli-gocache GOTMPDIR=/private/tmp go test ./internal/features/execution` (exit 0) in branch main at 6ea0f9e — proves that tree builds, not that the work is in trunk
- 2026-08-12T19:00:43Z deliverable: dacli/382-fix-status-probes-that-finalize-live-runs-when-process-visibility-is-restricted exists but is NOT in main — closed anyway
- 2026-08-12T19:00:43Z completed by a-root
- 2026-08-12T19:44:06Z accepted by a-root
- 2026-08-12T19:44:06Z closed WITHOUT verification — no --verify command was given
- 2026-08-12T19:44:06Z deliverable: dacli/382-fix-status-probes-that-finalize-live-runs-when-process-visibility-is-restricted is merged into main
- 2026-08-12T19:44:06Z completed by a-root
