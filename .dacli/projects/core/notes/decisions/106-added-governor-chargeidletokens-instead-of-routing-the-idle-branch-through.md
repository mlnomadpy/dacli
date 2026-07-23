---
id: d-106-added-governor-chargeidletokens-instead-of-routing-the-idle-branch-through
kind: note
note_kind: decision
created: 2026-07-23T14:56:53Z
created_by: a-j4dcnqkbat
about: [[106]]
---
# 106: added Governor.ChargeIdleTokens instead of routing the Idle branch through AfterCycle
## Chose
106: added Governor.ChargeIdleTokens instead of routing the Idle branch through AfterCycle
## Rejected
calling gov.AfterCycle(0, tokens) from the Idle branch to reuse the existing windowSpent writer
## Because
AfterCycle also increments the cycle counter and the zeroStreak thrash-guard counter -- an idle tick regenerates backlog, it is not a completed sprint, so folding it into AfterCycle would inflate Cycle() and could trip NoProgressHalt on an idling-but-healthy loop. A dedicated ChargeIdleTokens(tokens) method adds only windowSpent += tokens, keeping the thrash guard's signal (real cycles) separate from the window guard's signal (all token spend, idle included).
