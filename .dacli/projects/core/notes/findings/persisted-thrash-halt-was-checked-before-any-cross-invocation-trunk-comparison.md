---
id: f-persisted-thrash-halt-was-checked-before-any-cross-invocation-trunk-comparison
kind: note
note_kind: finding
created: 2026-08-12T15:30:56Z
created_by: a-codex-maintainer-2amnk2
about: "[[379]]"
severity: major
---
# Persisted thrash halt was checked before any cross-invocation trunk comparison
internal/features/orchestration/orchestration.go initializes the current trunk baseline only inside driver.loop, while the prior governor snapshot lacked a trunk marker; a restored zero_streak at the limit therefore could not distinguish unchanged trunk from a merge between invocations. Focused mutation failure: TestLoopRestartRecoversHaltedStreakAfterTrunkAdvance ran 0 cycles when ResetZeroStreak was disabled.
