---
id: f-100-calibrate-loop-advise-store-tokensperrun-implemented-both-acceptance
kind: note
note_kind: finding
created: 2026-07-23T12:57:22Z
created_by: a-cbbzr945ja
about: [[100]]
severity: moderate
---
# 100-calibrate: loop --advise + store.TokensPerRun implemented, both acceptance criteria met
Commit 51a136d by a-cbbzr945ja (fixer). Staged: internal/store/{calibration.go,calibration_test.go}, internal/features/orchestration/{orchestration.go,state_test.go}, docs/WALKTHROUGH.md, and the 100 task file. (1) store.TokensPerRun(samples, role) (internal/store/calibration.go) is new: median/p10/p90/n of raw output-token actuals grouped by Band.Role ALONE (not the full role x model x runtime Band the rest of calibration.go uses) since a future spawn has not picked a model/runtime yet — see the decision note. (2) dacli loop --advise (internal/features/orchestration/orchestration.go printLoopAdvisory) reports width*median(impl-role tokens/run) + median(review-role tokens/run) as the expected per-cycle token cost, with the same n>=10 AUTHORITATIVE/provisional gate every other calibrate readout uses; it short-circuits before the unbounded-loop stop-condition refusal and before the rw-grant check (pure read, no spawn), mirroring spawn --advise's contract. (3) Uses EXISTING calibration samples via store.CalibrationSamples -> store.TokensPerRun, no new data collection. go build ./... clean; go test ./internal/... all green, including new TestTokensPerRunGroupsByRoleAcrossModelsAndRuntimes, TestTokensPerRunUnknownRoleIsEmpty (store), and TestLoopAdviseReportsCalibratedCycleCostWithoutSpawning (orchestration, end-to-end through cmdLoop). NOTE: manual CLI smoke of the built binary was blocked by the headless sandbox (mkdir/exec outside the worktree needs approval), so verified via build+unit tests only, consistent with prior sibling tasks (027/028) hitting the same sandbox limit. Owner: verify and close via dacli task check 100 --n 1/--n 2, task done 100, then merge --task 100.
