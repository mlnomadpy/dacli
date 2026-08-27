---
id: f-task-494-inherited-task-492-s-stale-recovery-transfer-claim
kind: note
note_kind: finding
created: 2026-08-27T23:44:03Z
created_by: a-maintainer-e0s56a
about: "[[t-01M0N00V8CYZ3S125G5HJ2CTYN]]"
severity: major
---
# Task 494 inherited task 492's stale recovery-transfer claim
Run 01M12SP0EM8EDXNB8HVPPCYXSQ proc.txt records claims internal/store,internal/features/execution even though the defect is implemented in internal/features/vcs/vcs.go:129-153,491-500 and its required public regression belongs in internal/cli/vcs_test.go. agentClaims scans every transfer newest-first by owner alone, so task 492's historical root transfer became task 494's spawn claim. The current rw child cannot lawfully edit or commit the required files; root must correct the current run claim or re-spawn task 494 with exact claims internal/features/vcs,internal/cli. No --force was used.
