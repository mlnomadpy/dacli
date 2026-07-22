---
id: f-f2-complete-on-branch-dacli-041-spawn-advises-a-measured-token-budget-and-max
kind: note
note_kind: finding
created: 2026-07-22T17:46:15Z
created_by: a-q41r9cfexp
about: [[041]]
severity: major
---
# F2 complete on branch dacli/041 — spawn advises a measured token budget and --max-tokens gates the spawn
Commit e0aafbd by a-q41r9cfexp. Staged ONLY the 2 scoped files (git add + dacli commit --no-add): internal/store/calibration.go, internal/features/execution/execution.go. (1) store.MedianTokenRatio(samples, band) is the new F1->F2 primitive: median TokenRatio() over the band's HasTokens() samples + their count n (reuses spm.Median; store already imports spm, no cycle). (2) printAdvisory (execution.go) now LEADS with the measured token budget whenever the band has any token sample: ~median-output-tokens/point x Te suggested at n>=10, PROVISIONAL/no-firm-number at 1<=n<10; a band with zero token samples falls back to today's wall-clock advice unchanged (honest fallback). (3) new --max-tokens N flag: bandTokenBudget(w,t,band) computes expected = MedianTokenRatio x Te from the SAME samples the advisory shows; if expected EXCEEDS N the spawn refuses (clikit.Refusedf, exit 3) unless --force (loud stderr) — mirroring the D3 taint gate. n<10 => warn not refuse (provisional); no token history or unestimated task => note, proceed. A text-runtime spawn (no token data, no --max-tokens) still advises on wall-clock and NEVER refuses. go build ./... clean; go vet + gofmt clean; go test ./internal/... all green incl. internal/store, internal/features/execution, internal/cli (TestFeatureSlicesAreIsolated). MedianTokenRatio verified by throwaway store test (median/in-band-filter/no-token->(0,0)), not committed. Owner: verify and close via dacli task check/done 041 + dacli merge --task 041.
