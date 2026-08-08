---
id: f-051-complete-on-branch-dacli-051-commit-92238cc-all-4-acceptance-met-build-test
kind: note
note_kind: finding
created: 2026-07-22T16:26:30Z
created_by: a-8g6b17xcdq
about: [[051]]
severity: major
---
# 051 complete on branch dacli/051 — commit 92238cc, all 4 acceptance met, build+test green
Committed 92238cc by a-8g6b17xcdq on branch dacli/051-fix-insight-spm-brief-gates-blocked-task-consistency-gate-parallelizable. Staged the 5 required files (git add explicit + dacli commit --no-add --force, because calibration.go is required by acceptance 3 but sits outside my recorded insight/spm/brief/gates claim). ACCEPTANCE all satisfied: (1) BLOCKED CONSISTENCY: cmdCriticalPath (insight.go:~420) now excludes blocked from the schedule exactly like cmdNext, and BOTH edge loops now edge only to a scheduled node (openIDs) instead of any not-done dep, so an open task depending on a blocked task no longer makes spm.ComputeCPM fail 'edge references unknown task' — readiness against a blocked dep is still enforced by ready(). (2) DECISIONS GATE: gates.go case 'decisions' now counts only notes whose Rejected section is non-empty (not just len(notes)>=want), Why reports 'M of N recorded carry a rejection'; a real rejection may be terse ('async queue') so it is a non-empty check, not the 20-char unfilled bar. spm Network.Parallelizable doc (criticalpath.go:259) rewritten to state it does NOT filter by dependency readiness (a Network carries no edges) — the false 'dependencies already satisfied' claim is gone. (3) BRIEF: 'What siblings found' (brief.go:205) now honors MillerCap across notes+pending events with an announced 'N findings beyond the working-memory cap' omission and the trust-floor computed over shown findings only; calibration.go now walks RunsDir ONCE via runRecords (merging the old runBands+runUsage double walk), and cmdEstimate uses store.LoadCalibration (one walk backing both TaskBand and Samples) instead of TaskBand+CalibrationSamples (which was a 3rd walk). (4) go build ./... clean; go test ./internal/... all 14 packages green incl. cli TestStageGates, store, spm, brief. Owner: verify and close via dacli task check/done + dacli merge --task 051.
