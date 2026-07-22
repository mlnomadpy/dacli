---
id: f-073-complete-on-branch-dacli-073-all-4-acceptance-criteria-met-commit-f34f08a
kind: note
note_kind: finding
created: 2026-07-22T19:20:16Z
created_by: a-n2nn0v2g31
about: [[073]]
severity: moderate
---
# 073 complete on branch dacli/073-...; all 4 acceptance criteria met, commit f34f08a
Committed f34f08a by a-n2nn0v2g31 on branch dacli/073-fix-vcs-pr-rw-grant-check-security-pr-trust-floor-contrib-double-count. Staged ONLY 4 files (git add + dacli commit --no-add): internal/features/vcs/{lifecycle.go,vcs.go,lifecycle_test.go}, internal/cli/vcs_test.go. ACCEPTANCE: (1) SECURITY — cmdPR now refuses a non-rw grant BEFORE any gh call (lifecycle.go:135, 'opening a PR needs an rw grant'), matching push/merge/integrate; new TestPRRefusesReadOnlyGrant (cli) proves a ro agent gets exit-3. (2) cmdPR records the PR URL as model.EventComment, not model.EventFinding (lifecycle.go:~166) — a comment lands as a Log line and never syncs into a durable never-graded NoteFinding, so it no longer drags the brief trust-floor to unverified nor consumes a finding slot. (3) cmdContrib skips APPLIED EventFindings when counting against (vcs.go:~314, 'e.Against != "" && !e.Applied') so a ro reviewer's finding is counted once (as the pending event or its synced note, never both); new TestContribDoesNotDoubleCountSyncedFinding proves 1 finding(s)-against after sync (verified it reports 2 with the fix reverted). (4) Also deflaked pre-existing TestVerdictReviewRendersRecordedVerdicts (same-ms ULID tie) with a 2ms pause so the suite is deterministically green. go build ./... clean; go test ./internal/... all green (cli 5.7s, vcs green, verdict test green x15). Owner: verify and close via dacli task check/done + dacli merge --task 073.
