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
## Log
