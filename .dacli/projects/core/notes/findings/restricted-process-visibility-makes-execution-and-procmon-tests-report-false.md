---
id: f-restricted-process-visibility-makes-execution-and-procmon-tests-report-false
kind: note
note_kind: finding
created: 2026-08-12T16:07:26Z
created_by: a-codex-loop-auditor-21a3z2
about: "[[383]]"
severity: major
---
# Restricted process visibility makes execution and procmon tests report false product failures
With GOCACHE=/private/tmp/dacli-go-cache-383, go test ./... fails TestExecRuntimeDetachedDeliversAnOversizedPrompt at internal/features/execution/execruntime_test.go:331 with 0/164000 bytes, TestRunStillLivePreservesTask177AfterRuntimeLeaderExit at runstilllive_unix_test.go:36, and three procmon tests. The prompt test's awaitDetachedExit at execruntime_test.go:64 returns immediately whenever procmon.ProcState cannot observe the PID, conflating denied visibility with a reaped child and reading capture before the guardian runs. Focused count=3 reproduced the same 0-byte false failure all three times. Nearby execution tests already skip when process identity is unobservable, so the suite handles the same missing premise inconsistently.
