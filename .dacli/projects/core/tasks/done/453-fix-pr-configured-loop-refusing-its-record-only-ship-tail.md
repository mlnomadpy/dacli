---
id: t-01KZYXX2BZPRTWR5Z982B6SM76
kind: task
created: 2026-08-14T01:24:07Z
created_by: a-root
owner: a-root
github:
  issue: 663
  repo: mlnomadpy/dacli
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# Fix PR-configured loop refusing its record-only ship tail
## Context
Adopted from GitHub issue #663.

Implementation and claim boundary: `internal/features/orchestration` owns effective-policy propagation into `recordSelfPR`, its driver/landing-policy regression tests, and truthful loop output. Consume the existing ship policy contract without changing `internal/features/ship` or the shared landing model.

## Reproduction

1. Configure an existing linked project with landing.mode: pr and landing.base: main.
2. Run dacli loop --project core --max-cycles 1.
3. Let the cycle reach its record-only tail, including a cycle where worker spawns fail before a PR is opened.

Observed in cycle 92:

    record: ship failed: dacli: project landing policy requires the PR path; re-run with --pr, or explicitly override it with --landing-mode local

## Proven cause

internal/features/orchestration/orchestration.go recordSelfPR builds the record tail through shipArgs("--no-accept", "--no-integrate", "--project", project).

shipArgs forwards landing flags only when landingExplicit is true. A durable project policy is not an override, so the record command receives neither --pr nor --landing-mode pr. ship resolves the configured PR policy and then correctly refuses because the caller did not select its PR path.

This survived #662 because the tests cover explicit policy forwarding and landing checkpoints, but not a configured non-override PR policy reaching recordSelfPR.

## Manual workaround

None is safe for the normal loop. loop --no-pr would explicitly bypass the required GitHub landing policy, contrary to the project contract. The loop can continue worker execution, but its collaboration record cannot ship.

## Design

Make the loop record tail consume the already-resolved effective landing policy exactly like its landing path. In PR mode it must select the PR-capable ship path while preserving --no-accept and --no-integrate. The record branch must remain bookkeeping-only; it must not open task PRs or locally integrate code.

## Acceptance criteria

- [ ] A project configured with landing.mode=pr causes recordSelfPR to invoke a ship command accepted by the PR policy.
- [ ] The effective configured landing base is preserved without being mislabeled as a command-line override.
- [ ] The record tail remains --no-accept and --no-integrate and cannot locally land task code.
- [ ] A pending task PR still holds the record push; a cycle with no pending PR pushes main and dacli-record as before.
- [ ] Explicit loop --no-pr/local override retains its audited local behavior.
- [ ] A driver regression test reproduces the configured-policy failure before the fix and proves the corrected record command.
- [ ] Loop dry-run and operator output remain truthful about the effective policy and record action.
- [ ] Focused orchestration tests and go test ./... pass.

## Acceptance
- [x] A project configured with landing.mode=pr causes recordSelfPR to invoke a ship command accepted by the PR policy.
- [x] The effective configured landing base is preserved without being mislabeled as a command-line override.
- [x] The record tail remains --no-accept and --no-integrate and cannot locally land task code.
- [x] A pending task PR still holds the record push; a cycle with no pending PR pushes main and dacli-record as before.
- [x] Explicit loop --no-pr/local override retains its audited local behavior.
- [x] A driver regression test reproduces the configured-policy failure before the fix and proves the corrected record command.
- [x] Loop dry-run and operator output remain truthful about the effective policy and record action.
- [x] Focused orchestration tests and go test ./... pass.
## Log
- 2026-08-14T01:24:54Z claimed by a-junior-ap0nav
- 2026-08-14T01:26:14Z claimed by a-root
- 2026-08-14T01:27:18Z blocked: Prior routed worker was not authenticated and produced no work; reset ownership before Codex retry.
- 2026-08-14T01:27:18Z reopened by a-root: Authentication failure is resolved by explicitly routing this retry to the configured Codex maintainer role. (cleared 0 acceptance box(es) — the close claimed work that was not verified)
- 2026-08-14T01:46:08Z accepted by a-root
- 2026-08-14T01:46:08Z verified by `GOCACHE=/tmp/dacli-accept-453 go test ./...` (exit 0) in branch main at 8dee8c2 — proves that tree builds, not that the work is in trunk
- 2026-08-14T01:46:08Z deliverable: dacli/453-fix-pr-configured-loop-refusing-its-record-only-ship-tail is merged into main
- 2026-08-14T01:46:08Z completed by a-root
- 2026-08-14T01:46:43Z accepted by a-root
- 2026-08-14T01:46:43Z closed WITHOUT verification — no --verify command was given
- 2026-08-14T01:46:43Z deliverable: dacli/453-fix-pr-configured-loop-refusing-its-record-only-ship-tail is merged into main
- 2026-08-14T01:46:43Z completed by a-root
- 2026-08-14T01:54:55Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/664 (event 01KZYYR4GJRY4A0A4RPN04STPV)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-453 go test ./...","exit_code":0,"duration_ms":58179,"artifact_hash":"sha256:28bbcb42e75f6ef37087b39b42ca83561ab496d1d2720bc35bd103c46a13041a","verifier":"a-root","branch":"main","commit_sha":"8dee8c29379c6eb7b59ecfb16b8aae52597cc46d"}
