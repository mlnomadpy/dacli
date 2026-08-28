---
id: t-01M12QX9HEPKAAS1033W6HS45D
kind: task
created: 2026-08-27T23:12:03Z
created_by: a-root
owner: a-root
github:
  issue: 841
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# [agent-report] [agent-report] integrate and accept enforce contradictory PR landing order
## Context
Adopted from GitHub issue #841.

Symptom: after PR 840 had every required GitHub check green, dacli integrate --tasks 509 --pr --merge refused because task 509 was open. Running dacli accept 509 with successful full verification then refused because the PR branch was not yet in main. The normal commands therefore form a cycle: integrate requires done, while accept requires landed. The documented playbook also requires merge plus fresh-trunk inspection before acceptance and issue closure. Suspected cause: integrate validates task status before the PR landing transaction, while accept applies its unlanded guard without recognizing a checks-passing mergeable PR or transaction context. Manual step: explicitly accept with --allow-unlanded, immediately integrate the green PR, fast-forward main, and rerun go test ./... on fresh trunk. Expected design: a single transaction such as ship --tasks should accept with deferred landing, merge only a checks-passing PR, inspect fresh trunk, record the landing verdict, and close the task; its dry-run and execution should support an explicitly selected open task without pre-rejecting it as not done. Acceptance: reproduce with a green open PR and an open fully checked task; prove the ordinary non-force path completes the transaction; prove merge failure leaves the task non-final or rolls it back; prove GitHub issue closure happens only after merged state and fresh-trunk verification; cover direct integrate/accept recovery messaging so neither command recommends an impossible next command.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] A regression fixture starts with an open, fully checked task and a green mergeable PR; the ordinary non-force ship path completes the transaction without requiring contradictory manual commands.
- [ ] A PR merge/check failure leaves the task nonterminal or restores it transactionally, preserving the branch, worktree, evidence, and actionable recovery state.
- [ ] The linked GitHub issue and local task close only after GitHub reports merged, fresh configured trunk contains the exact reviewed head/tree, and post-landing verification succeeds.
- [ ] Direct `integrate` and `accept` refusals recommend a valid recovery command and never direct the operator into the same accept-before-merge/merge-before-accept cycle.
## Log
- 2026-08-28T13:42:00Z a-root: materialized the explicit acceptance sentence from GitHub issue #841 after adoption left the section empty; tracked general migration in #875.
- 2026-08-28T13:41:04Z claimed by a-maintainer-wg7dnx
