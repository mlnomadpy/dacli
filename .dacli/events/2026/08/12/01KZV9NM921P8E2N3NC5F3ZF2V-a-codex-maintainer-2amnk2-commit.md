---
id: 01KZV9NM921P8E2N3NC5F3ZF2V
kind: event
event_kind: commit
created: 2026-08-12T15:32:49Z
created_by: a-codex-maintainer-2amnk2
about: "[[t-01KZV5RSPJ5PSG74VHH2Q4A7JQ]]"
origin: agent
applied: true
---
dec221d 379: recover halted loop when trunk advances

Persist the measured trunk marker with governor state and clear only the no-progress streak when a newer marker is observed. Preserve the halt when trunk is unchanged and surface automatic versus explicit-reset recovery in loop status.

Red mutation: TestLoopRestartRecoversHaltedStreakAfterTrunkAdvance failed with “trunk advancement must clear the persisted halt and run one cycle, ran 0” when ResetZeroStreak was disabled.
role: codex-maintainer
