---
id: t-01KZYQ5ECBFAX3KD08T3VQ5KWV
kind: task
created: 2026-08-13T23:26:22Z
created_by: a-root
owner: a-root
depends_on: [450, 449, 448]
github:
  issue: 637
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
---
# [agent-report] Add an enforceable PR-only landing policy for linked GitHub projects
## Context
Adopted from GitHub issue #637.

A GitHub-linked project can currently complete tasks with local 'dacli integrate --into dev', and the normal loop does not require PR review unless the operator remembers '--pr'. Please add a project-level landing policy such as landing.mode=pr and landing.base=dev. When enabled, integrate/ship/loop should refuse local-only merges, push the task branch, open or reuse a PR, wait for required checks/review, and record the PR URL before marking the task landed. The policy should survive loop cycles and make any fallback to local integration an explicit error or override, not the default.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] Project configuration can persist and display a validated `landing.mode=pr` and `landing.base=<branch>`; existing projects default to current local behavior.
- [ ] `integrate`, `ship`, and `loop` consume the same resolved landing policy instead of independently interpreting flags.
- [ ] In PR mode, a local-only landing attempt is refused before task or git state mutates unless the operator supplies a documented explicit override.
- [ ] PR mode pushes the canonical task branch and opens or reuses one PR targeting the configured base.
- [ ] Required checks and review requirements are observed before the task is recorded as landed.
- [ ] The PR URL and landing outcome are recorded in task/event history before terminal status is materialized.
- [ ] Dry-run reports the exact effective landing mode, base, override, PR action, and gates without mutation.
- [ ] Restarting a loop cycle is idempotent: it reuses the existing branch/PR and does not duplicate state.
- [ ] Unit and integration tests cover local default, enforced PR mode, refusal, explicit override, PR reuse, failed checks, and successful landing.
- [ ] User and operator documentation explains configuration, precedence, recovery, and GitHub prerequisites.
## Log
- 2026-08-13T23:58:44Z blocked: 
