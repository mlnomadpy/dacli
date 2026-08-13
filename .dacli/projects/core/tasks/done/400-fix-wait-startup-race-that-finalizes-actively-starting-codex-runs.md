---
id: t-01KZVSGRZ34ZAGRMDEP64KYKRE
kind: task
created: 2026-08-12T20:09:47Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 535
  repo: mlnomadpy/dacli
---
# Fix wait startup race that finalizes actively starting Codex runs
## Acceptance
- [x] Given two freshly detached Codex runs whose transcripts are already advancing, dacli wait does not finalize either run as no visible result during process-registration startup
- [x] A regression reproduces runs 01KZVSF64J and 01KZVSFBXR: wait invoked 7-12 seconds after spawn leaves both runs running while their worker processes or transcripts are live
- [x] The startup grace and liveness check are bounded, observable, and still finalize a genuinely dead detached launch
- [x] dacli agents, runs list, and wait agree on each run lifecycle state throughout startup and completion
## Log
- 2026-08-12T20:17:09Z claimed by a-codex-maintainer-w6vv23
- 2026-08-12T20:37:57Z accepted by a-root
- 2026-08-12T20:37:57Z verified by `GOCACHE=/private/tmp/dacli-400-accept-gocache go test -race ./internal/features/execution -count=1` (exit 0) in branch main at 20d28e1 — proves that tree builds, not that the work is in trunk
- 2026-08-12T20:37:57Z deliverable: dacli/400-fix-wait-startup-race-that-finalizes-actively-starting-codex-runs is merged into main
- 2026-08-12T20:37:57Z completed by a-root
- 2026-08-12T20:40:43Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/541 (event 01KZVTDWQ966839ZNPPMXB8GTF)
- 2026-08-12T20:40:43Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/541 at merge commit 20d28e15f252aefb5845b38f54580c67015ca904 into main; local cleanup complete (event 01KZVV1W768T5QVEGMMT2RPZDZ)
