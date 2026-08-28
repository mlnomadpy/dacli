---
id: t-01M147RA2EPR7BKWWZ3WS8CY44
kind: task
created: 2026-08-28T13:08:11Z
created_by: a-root
owner: a-root
github:
  issue: 868
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
depends_on: "[t-01M146BA434CD4A9778E8BNJ61, t-01M147RA4B2C2NAH855VBNT2SJ, t-01M146BA07Z5BTS3TTB2ADW7D4]"
---
# Make independent review and bounded correction a resumable phase
## Context
Adopted from GitHub issue #868.

## Parent

Part of #864. Complements #857 verification binding and #859 resumable loop recovery.

## Observed symptom

Independent review may be configured but unenforceable because the runtime cannot provide the requested read-only/isolation contract. Findings are durable, but there is no single bounded correction transaction that binds a structured review finding to the corrected tree and re-review.

## Objective

Make independent review an enforceable, structured delivery phase with bounded correction turns.

## Required result contract

- Verdict: approve, request-changes, inconclusive, no-response, or infrastructure-failure.
- Severity, stable finding ID, file, line/range, evidence, affected invariant, and suggested verification.
- Reviewer identity, role/runtime/model, independence relationship, commit/tree SHA, and observed PR generation.



## Non-goals

- Requiring cross-provider review for every task.
- Granting a read-only reviewer write authority to implement fixes.
- Infinite correction loops.

## Manual workaround today

Operators inspect findings, manually resume an implementation agent, then separately spawn another reviewer and compare commits.

## Acceptance
- [ ] Preflight proves the selected reviewer runtime can enforce its declared grant/isolation and return the structured result before implementation starts.
- [ ] No-response, inconclusive, and infrastructure-failure never count as approval or evidence-bearing refutation.
- [ ] A request-changes result starts at most the configured number of correction turns, each bound to the finding IDs and exact prior tree.
- [ ] Re-review observes the corrected commit/tree and cannot approve stale code.
- [ ] Correction work reuses or safely recovers the canonical task worktree/branch with claims and attribution intact.
- [ ] GitHub review state and line comments are produced from the same structured findings without leaking private workspace evidence.
- [ ] Mutation tests fail when review is skipped after correction, a stale tree is approved, or missing output is treated as success.
## Log
- 2026-08-28T13:32:48Z dependency edit by a-root (event 01M1495CBNJZ2R9QK04NK3JMRC)
