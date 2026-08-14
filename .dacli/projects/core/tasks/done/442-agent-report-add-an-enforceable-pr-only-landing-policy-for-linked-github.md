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
- [x] Project configuration can persist and display a validated `landing.mode=pr` and `landing.base=<branch>`; existing projects default to current local behavior.
- [x] `integrate`, `ship`, and `loop` consume the same resolved landing policy instead of independently interpreting flags.
- [x] In PR mode, a local-only landing attempt is refused before task or git state mutates unless the operator supplies a documented explicit override.
- [x] PR mode pushes the canonical task branch and opens or reuses one PR targeting the configured base.
- [x] Required checks and review requirements are observed before the task is recorded as landed.
- [x] The PR URL and landing outcome are recorded in task/event history before terminal status is materialized.
- [x] Dry-run reports the exact effective landing mode, base, override, PR action, and gates without mutation.
- [x] Restarting a loop cycle is idempotent: it reuses the existing branch/PR and does not duplicate state.
- [x] Unit and integration tests cover local default, enforced PR mode, refusal, explicit override, PR reuse, failed checks, and successful landing.
- [x] User and operator documentation explains configuration, precedence, recovery, and GitHub prerequisites.
## Log
- 2026-08-13T23:58:44Z blocked: 
- 2026-08-14T01:18:19Z reopened by a-root: All three dependency slices (tasks 448, 449, and 450) are merged and accepted; reconcile the umbrella against their aggregate acceptance criteria. (cleared 0 acceptance box(es) — the close claimed work that was not verified)
- 2026-08-14T01:19:33Z accepted by a-root
- 2026-08-14T01:19:33Z verified by `GOCACHE=/tmp/dacli-accept-442 go test ./...` (exit 0) in branch main at 3ab6530 — proves that tree builds, not that the work is in trunk
- 2026-08-14T01:19:33Z deliverable: no dacli/442-agent-report-add-an-enforceable-pr-only-landing-policy-for-linked-github branch — nothing to check against main
- 2026-08-14T01:19:33Z completed by a-root
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-442 go test ./...","exit_code":0,"duration_ms":68146,"artifact_hash":"sha256:a8500ebc02c4599842ed9e36f5f83cba1a1487dd8d073a695e6f847d81c845ab","verifier":"a-root","branch":"main","commit_sha":"3ab6530159a4ef52826d816188574c3d07100a20"}
