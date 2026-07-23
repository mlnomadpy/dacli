---
id: f-093-complete-on-branch-dacli-093-loop-status-implemented-both-acceptance
kind: note
note_kind: finding
created: 2026-07-23T10:43:41Z
created_by: a-wcs10h3a5b
about: [[093]]
severity: moderate
---
# 093 complete on branch dacli/093 — loop status implemented, both acceptance criteria met
Commit 23229fc by a-wcs10h3a5b (fixer). Staged: internal/features/orchestration/{orchestration.go,state.go,state_test.go}, .dacli/.gitignore. (1) dacli loop status prints cycle count, trunk marker, tokens spent this window, and ready backlog size — orchestration.go cmdLoopStatus (new 'loop status' Command). (2) State is persisted via writeLoopState/readLoopState (state.go) as a key:value text file at .dacli/loop/<project>.txt (added to .dacli/.gitignore alongside runs/build/worktrees), written at every governor checkpoint in driver.loop() (both the pre-cycle Before() decision and the post-cycle AfterCycle() decision) via the new driver.saveState helper, tracking a new driver.lastTrunkMarker field. Covered by TestLoopPersistsStateForStatusToRead (runs the driver end-to-end, reads the persisted state via readLoopState, then also drives cmdLoopStatus directly and asserts the printed output) and TestLoopStatusErrorsWithoutAPriorLoopRun (no persisted state -> real error, not zeroes). go build ./... clean; go test ./internal/... all green. NOTE: full governor round-trip resume across restarts (window/streak) is out of scope here — that is task 096, a separate open task; this task only persists a read-only status snapshot. Owner: verify and close via dacli task check 093 --all + dacli task done 093.
