---
id: t-01M13S7VDH9ZN15AJAEYS5QFC4
kind: task
created: 2026-08-28T08:54:32Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 847
  repo: mlnomadpy/dacli
---
# Fail runtime launch when its durable transcript cannot be created
## So that
every reported agent launch has the durable transcript required for wait, usage, recovery, and audit truth
## Acceptance
- [x] Foreground execRuntime returns a contextual transcript-creation error before starting the runtime when transcriptPath cannot be created.
- [x] Detached execRuntime returns the same fail-closed error and records no live process identity.
- [x] Focused regressions prove the runtime binary is never invoked on transcript creation failure.
- [x] Mutation evidence and go test ./internal/features/execution pass.
## Log
- 2026-08-28T09:51:46Z claimed by a-fixer-64tsev
- 2026-08-28T09:56:32Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/850 (event 01M13WS8TCFW08KGV38AK85HG9)
- 2026-08-28T10:19:24Z accepted by a-root
- 2026-08-28T10:19:24Z verified by `env GOCACHE=/tmp/dacli-gocache-531-accept go test ./internal/features/execution -count=1` (exit 0) in branch dacli/531-fail-runtime-launch-when-its-durable-transcript-cannot-be-created at 3e93aac6 — proves that tree builds, not that the work is in trunk
- 2026-08-28T10:19:24Z deliverable: dacli/531-fail-runtime-launch-when-its-durable-transcript-cannot-be-created exists but is NOT in main — closed anyway
- 2026-08-28T10:19:24Z completed by a-root
- 2026-08-28T10:28:06Z a-root: Landing policy override: mode=pr base=main (event 01M13Y38MYAYK3MHF666HMKJYZ)
- 2026-08-28T10:28:06Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/850 at merge commit 9aeb8d9948fbc97c13625631f0e575ece2def935 into main (generation 0) (event 01M13Y3FN5NMQ84QQPQZMVXD31)
- 2026-08-28T10:28:06Z a-root: Landing policy override: mode=pr base=main (event 01M13Y42ZJPCVQQ7PNWWMW3ZJ3)
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-gocache-531-accept go test ./internal/features/execution -count=1","exit_code":0,"duration_ms":36780,"artifact_hash":"sha256:a3a81442d05a7cc6486f58ee5a07462185fb3d0d3cce641fb648fb7365966f04","verifier":"a-root","branch":"dacli/531-fail-runtime-launch-when-its-durable-transcript-cannot-be-created","commit_sha":"3e93aac6b7f3473ce1096dc85bddedebbaa05601"}
