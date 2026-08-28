---
id: t-01M147RA4B2C2NAH855VBNT2SJ
kind: task
created: 2026-08-28T13:08:11Z
created_by: a-root
owner: a-root
github:
  issue: 867
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
depends_on: "[t-01M12THP840GDS4BTS539D2FQF]"
---
# Preflight the complete loop and enforce role capacity consistently
## Context
Adopted from GitHub issue #867.

## Parent

Part of #864. Related harness-policy regression: #845.

## Observed symptom

`team assign` can refuse an oversized task while a loop with an explicit implementation role previews or attempts that same under-capacity assignment. Separately, an impossible reviewer runtime can be discovered only after implementation and retried every cycle.

## Objective

Preflight the complete resolved cycle before implementation and make role-capacity overrides explicit, reasoned, and durable.

## Required preflight

- Every selected implementation task/role/model/runtime/grant/claim combination.
- Reviewer runtime, grant/isolation, model, token controls, and required output contract.
- Landing base, GitHub authentication/observability, and configured PR/check policy.
- Verification command/profile availability and working directories.
- WIP, claims, rolling/cycle budgets, timeouts, and STOP state.



## Non-goals

- Treating every token ceiling as enforceable when a runtime declares it advisory.
- Automatically switching harness families.
- Repeatedly retrying exit-code 3 refusals.

## Manual workaround today

Operators separately run role assignment and runtime preflight, inspect loop preview, and manually replace impossible roles.

## Acceptance
- [ ] The same task and resolved role receive the same capacity verdict from `team assign`, start preview, and live loop selection.
- [ ] An explicit under-capacity role is refused before any worker starts unless an owner supplies a reasoned override recorded with task, role, capacity delta, actor, and expiry/invocation scope.
- [ ] An impossible reviewer preflight halts before implementation when review is required; it is not retried unchanged in later cycles.
- [ ] Harness pinning applies to implementation, review, recovery, and fallback; a single-Codex run never silently selects Claude.
- [ ] Preflight emits one versioned JSON result listing every phase, verdict, evidence, and remediation.
- [ ] Transient external failures remain retryable and distinct from permanent capability/policy refusals.
- [ ] Mutation tests fail when explicit loop role selection bypasses capacity or when reviewer preflight is deferred until after implementation.
## Log
- 2026-08-28T13:32:48Z dependency edit by a-root (event 01M1495CFWTG0X2PPFHZR6HNH8)
