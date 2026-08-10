---
id: 01KZNZ2DHKN4BQGM97NJZFR299
kind: event
event_kind: finding
created: 2026-08-10T13:51:21Z
created_by: a-go-auditor-c2r8as
about: "[[t-01KZNYJ7P67QTZ0B65JPCZN47C]]"
origin: agent
applied: true
---
Malformed compound assertion silences the exact regression it names (guards_test.go:365)

internal/features/orchestration/guards_test.go:365, in TestGovernorUnmeasuredCycleLeavesThrashStreakAlone. The named invariant is 'an unmeasured cycle must leave the thrash streak at 1'. Setup: g.AfterCycle(0,10) makes the zero-progress streak = 1, and NoProgressHalt = 2. The assertion is: if d, why := g.AfterCycleUnmeasured(10); d != Halt && g.ZeroStreak() != 1 { t.Fatalf(...) }. PROOF it cannot fail for its stated bug (pure boolean logic, no execution needed): the Fatalf requires BOTH d != Halt AND streak != 1. The precise regression this test targets is AfterCycleUnmeasured wrongly treating the cycle as a real zero-progress cycle: that would increment the streak to 2, and since 2 == NoProgressHalt it returns d == Halt. Then 'd != Halt' is FALSE, the conjunction is FALSE, and line 365 stays SILENT on exactly that regression. A conjunction can only make a Fatalf guard WEAKER; the correct assertion is 'if g.ZeroStreak() != 1' alone (which line 371 already has correctly). CLASSIFICATION: weak assertion (malformed guard), NOT missing coverage. SEVERITY minor because the same bug is caught redundantly by line 368 (d == Halt -> Fatal) and line 371 (streak != 1 -> Fatalf), so no regression actually escapes the suite today; the fix is to simplify line 365 to enforce the invariant its message claims, so a future edit that removes 368/371 does not silently lose the check. This is the ONLY provably-vacuous assertion found across the packages audited.
