---
id: f-091-complete-on-branch-dacli-091-commit-b8b7dcb-both-acceptance-criteria-met
kind: note
note_kind: finding
created: 2026-07-23T10:23:49Z
created_by: a-s56ztpjbv1
about: [[091]]
severity: moderate
---
# 091 complete on branch dacli/091 — commit b8b7dcb — both acceptance criteria met
Committed b8b7dcb by a-s56ztpjbv1 (fixer) via git add + dacli commit --no-add, staging ONLY the 3 intended files: internal/features/orchestration/orchestration.go, internal/features/orchestration/driver_test.go, internal/store/calibration.go. ACCEPTANCE: (1) runCycle (orchestration.go) now snapshots store.LatestRunID(w) before running its phases and, via a deferred close, sets its named return to store.RunsTokensSince(w, since) — the real sum of output_tokens from every usage.txt written under RunsDir since that cursor, covering both the build wave's spawns and the review spawn. Two new exported store/calibration.go helpers (LatestRunID, RunsTokensSince) reuse the existing unexported readUsage parser so the semantics match what calibration already trusts; the Governor's AfterCycle/Before token-window logic was already correct (governor_test.go covers it), the gap was purely that runCycle always returned a hardcoded 0. (2) New driver_test.go:TestRunCycleSumsRealUsageTokensAndGovernorSleeps adds a usageRunner fake that, on each simulated 'spawn' call, writes a real RunsDir entry + usage.txt (mirroring execution.go's writeUsage), then asserts runCycle's real returned token sum (>= 1000 from 2 spawns at 500 each) causes gov.Before to return SleepWindow against a WindowTokens:100 budget — i.e. real per-cycle tokens, not a hand-fed number, trip the governor. go build ./... clean; go vet ./... clean; go test ./internal/... all green (incl. new test). Owner: verify and close via dacli task check/done + dacli merge --task 091.
