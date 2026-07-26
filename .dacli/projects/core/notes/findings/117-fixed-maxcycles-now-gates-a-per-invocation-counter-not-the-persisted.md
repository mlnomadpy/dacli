---
id: f-117-fixed-maxcycles-now-gates-a-per-invocation-counter-not-the-persisted
kind: note
note_kind: finding
created: 2026-07-26T22:51:06Z
created_by: a-7eanj1ps0j
about: [[117]]
severity: moderate
---
# 117: fixed — MaxCycles now gates a per-invocation counter, not the persisted cumulative one
internal/features/orchestration/governor.go: added unexported cyclesThisRun (never persisted/restored) alongside the existing persisted cumulative cycle. Governor.Before (governor.go:121) now checks cyclesThisRun>=MaxCycles instead of the restored cycle; AfterCycle (governor.go:157) increments both. Restore() deliberately leaves cyclesThisRun at 0 so a fresh process always gets its full --max-cycles budget regardless of the restored total, while cycle/windowSpent/windowStart/zeroStreak keep resuming exactly as before (status reporting, --window-tokens, --no-progress-halt all unaffected). Added TestGovernorMaxCyclesGatesOnThisRunNotRestoredTotal (governor_test.go) reproducing the exact collision at the Governor API level, and TestRepeatedBoundedInvocationsDoNotHaltOnPersistedTotal (state_test.go) driving 3 fresh --max-cycles 1 invocations end-to-end through driver.loop() the way the reporter's build-itself sprint program does. Full go build ./... and go test ./... green.
