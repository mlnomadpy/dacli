---
id: t-01KZYRZPW14R49ED19F0MPV11X
kind: task
created: 2026-08-13T23:58:11Z
created_by: a-root
owner: a-root
depends_on: [450]
github:
  issue: 655
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Enforce project PR landing policy in integrate and ship
## Context
Adopted from GitHub issue #655.

## Parent and dependency

Part of #637. Depends on the typed policy foundation issue created alongside this issue.

## Scope

Consume the resolved policy in `integrate` and `ship`. Keep task state transitions atomic around refusal and GitHub failures. Loop-specific orchestration and documentation belong to the follow-up slice.

Implementation and claim boundary: `internal/features/vcs` owns integrate/PR landing behavior and its tests; `internal/features/ship` owns ship composition and dry-run parity. Consume the shared policy foundation landed by task 450; do not redesign it in this slice.

## Acceptance criteria

- [ ] `integrate` and `ship` consume the shared effective landing policy rather than independently deriving mode or base.
- [ ] In `pr` mode, omitting the PR path is refused before git, task, event, or GitHub mutation.
- [ ] The documented explicit local override is recorded in output and durable event history.
- [ ] PR mode pushes the canonical task branch and opens or reuses one PR targeting the configured base.
- [ ] Required checks and review requirements must pass before the task is recorded as landed.
- [ ] A failed push, PR operation, check, or review gate leaves the task recoverable and not falsely landed.
- [ ] The PR URL and successful landing outcome are recorded before terminal task status is materialized.
- [ ] Dry-run reports effective mode, base, override state, PR action, and gates without mutation.
- [ ] Integration tests cover refusal, override, PR reuse, gate failure, successful landing, and dry-run parity.
- [ ] Focused package tests and `go test ./...` pass.

## Acceptance
- [x] `integrate` and `ship` consume the shared effective landing policy rather than independently deriving mode or base.
- [x] In `pr` mode, omitting the PR path is refused before git, task, event, or GitHub mutation.
- [x] The documented explicit local override is recorded in output and durable event history.
- [x] PR mode pushes the canonical task branch and opens or reuses one PR targeting the configured base.
- [x] Required checks and review requirements must pass before the task is recorded as landed.
- [x] A failed push, PR operation, check, or review gate leaves the task recoverable and not falsely landed.
- [x] The PR URL and successful landing outcome are recorded before terminal task status is materialized.
- [x] Dry-run reports effective mode, base, override state, PR action, and gates without mutation.
- [x] Integration tests cover refusal, override, PR reuse, gate failure, successful landing, and dry-run parity.
- [x] Focused package tests and `go test ./...` pass.
## Log
- 2026-08-14T00:20:02Z claimed by a-maintainer-2vktb5
- 2026-08-14T00:34:14Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/659 (event 01KZYTPEPCKKXHHT5HDKW1NNPC)
- 2026-08-14T00:35:20Z accepted by a-root
- 2026-08-14T00:35:20Z verified by `GOCACHE=/tmp/dacli-accept-449 go test ./...` (exit 0) in branch main at 59249fd — proves that tree builds, not that the work is in trunk
- 2026-08-14T00:35:20Z deliverable: dacli/449-enforce-project-pr-landing-policy-in-integrate-and-ship is merged into main
- 2026-08-14T00:35:20Z completed by a-root
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-449 go test ./...","exit_code":0,"duration_ms":61605,"artifact_hash":"sha256:1fc7ed4594145e842a6a1abdcb33c39265937e3c32adf1affb1b69cc19130f00","verifier":"a-root","branch":"main","commit_sha":"59249fded1fc2507bc24a01e729eb94b09d1ef67"}
