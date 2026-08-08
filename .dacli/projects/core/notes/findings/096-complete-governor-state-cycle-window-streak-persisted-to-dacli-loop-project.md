---
id: f-096-complete-governor-state-cycle-window-streak-persisted-to-dacli-loop-project
kind: note
note_kind: finding
created: 2026-07-23T12:16:06Z
created_by: a-5q01aq7dad
about: [[096]]
severity: moderate
---
# 096 complete: governor state (cycle/window/streak) persisted to .dacli/loop/<project>-governor.txt and reloaded by cmdLoop on start
Commit fefc8c6. New file internal/features/orchestration/state.go: writeGovernorState/readGovernorState persist Governor.State() (cycle, windowStart, windowSpent, zeroStreak) to a file distinct from the existing loopState status snapshot (that one is explicitly never consulted by loop control flow, per its doc comment; the new governorStateFile IS consulted). governor.go:65-99 adds Governor.State()/Restore() plus WindowStart()/ZeroStreak() getters. orchestration.go: driver.saveState now also calls writeGovernorState at every checkpoint (same call sites as the existing status snapshot); cmdLoop calls readGovernorState+gov.Restore before the loop starts, so a restarted process resumes cycle count, in-window token spend, and the thrash-guard streak instead of resetting them. Tests: TestGovernorStateRoundTrips (state.go round-trip incl. restored streak still tripping NoProgressHalt), TestGovernorStateAbsentIsHonestError (degrade path mirrors readLoopState), TestLoopRestartResumesGovernorState (two driver.loop() runs across a simulated restart — second process advances to cycle 2, not back to 1). go build ./... clean; go test ./internal/... all green.
