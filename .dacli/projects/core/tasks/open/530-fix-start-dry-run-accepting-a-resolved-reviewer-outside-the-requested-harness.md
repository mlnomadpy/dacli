---
id: t-01M12THP840GDS4BTS539D2FQF
kind: task
created: 2026-08-27T23:58:08Z
created_by: a-adversarial-reviewer-rvb458
owner: a-root
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 845
  repo: mlnomadpy/dacli
---
# Fix start dry-run accepting a resolved reviewer outside the requested harness
## Acceptance
- [ ] A regression fixture with stack-matching reviewers on two harnesses shows start --profile loop --harness <selected> --dry-run and live loop argument resolution select the same reviewer on the selected harness
- [ ] Operating-profile role and harness resolution fails closed before rendering when no compatible implementation or review role exists
- [ ] go test ./internal/features/orchestration passes, and the regression fails when harness-aware role resolution is removed
## Log
- 2026-08-28T00:26:56Z takeover by a-root from a-adversarial-reviewer-rvb458 (recovery: task takeover --force; reason: Recovering an evidence-backed task created by a retired adversarial reviewer so GitHub can remain the authoritative backlog)
