---
id: f-task-removal-fails-open-on-unreadable-live-run-evidence
kind: note
note_kind: finding
created: 2026-08-26T13:45:12Z
created_by: a-adversarial-reviewer-gstekv
about: "[[t-01KZXRP56538HNHDP3P4FJHGGP]]"
severity: major
---
# Task removal fails open on unreadable live-run evidence
internal/store/store.go:1217-1233 OwnerHasLiveRun returns false when .dacli/runs cannot be read and skips every malformed proc.txt; internal/store/remove.go:280-297 liveClaimants independently returns nil/skips the same evidence. Trigger: read permission drift on the runs directory or a partial proc.txt while read-write root runs task rm --force for a child-owned task. Wrong outcome: internal/features/planning/planning.go:994 treats false as proof the owner is not live, then RemoveTask can delete the task even though a live owner/claimant may be hidden. Existing internal/features/planning/reopen_test.go:259 covers only a readable valid live record. Recovery/takeover already fails closed via OwnerTaskHasRecoveryLease, so removal disagrees with that safety policy.
