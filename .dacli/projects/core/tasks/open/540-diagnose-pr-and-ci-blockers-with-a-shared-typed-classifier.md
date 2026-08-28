---
id: t-01M146BA26TEH86YE16XHDZKGY
kind: task
created: 2026-08-28T12:43:36Z
created_by: a-root
owner: a-root
github:
  issue: 858
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
depends_on: "[t-01M11HZW8X678270CFWDBK7YTN]"
---
# Diagnose PR and CI blockers with a shared typed classifier
## Context
Adopted from GitHub issue #858.

## Parent

Part of #855.

## Objective

Add native, structured PR and CI diagnosis:

```bash
dacli pr diagnose --task <ref>
dacli pr diagnose --task <ref> --json
```

The classifier must be reusable by reconciliation and loop recovery rather than implemented only in the command handler.

## Required classifications

- Test/workflow job failure, including failing annotations.
- Workflow syntax/configuration failure.
- Runner unavailable or queued beyond a documented threshold.
- Billing/spending/account restriction.
- GitHub authentication/authorization failure.
- GitHub network/API outage or rate limiting.
- Pending environment/reviewer approval.
- Merge conflict, stale/behind base, closed-unmerged PR, missing canonical PR, and superseded PR generation.
- Unknown when GitHub does not provide enough evidence.



## Non-goals

- Retrying or overriding failed required checks.
- Merging a PR.
- Repairing GitHub billing or credentials.

## Manual workaround today

Operators combine `dacli pr status`, `gh pr checks`, workflow-run inspection, annotations, branch comparison, and GitHub account status manually.

## Acceptance
- [ ] Table-driven fixtures for every required class produce a stable machine code, evidence/source fields, retryability, and an actionable next step.
- [ ] Check-suite annotations and workflow-run conclusions are inspected where available; diagnosis does not rely only on the aggregate `gh pr checks` exit code.
- [ ] Authentication, outage, permission, and unknown responses are distinguishable and never reported as test failure or success.
- [ ] Multiple PR generations are resolved against the canonical task head rather than the first historical PR.
- [ ] Human and JSON output are rendered from one typed result model shared below feature slices.
- [ ] Tests fail when a billing/auth/outage fixture is collapsed into generic red CI.
- [ ] The command is declared in the central command registry with accurate JSON/mutation/usage metadata and corresponding CLI/MCP contract coverage.
## Log
- 2026-08-28T12:44:47Z dependency edit by a-root (event 01M146DESEJWD3R7V69QYQAF3Z)
