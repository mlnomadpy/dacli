---
id: f-recovery-takeover-fails-open-when-run-state-is-unreadable
kind: note
note_kind: finding
created: 2026-08-26T12:56:39Z
created_by: a-adversarial-reviewer-wzq9fh
about: "[[t-01KZXRP56538HNHDP3P4FJHGGP]]"
severity: major
---
# Recovery takeover fails open when run state is unreadable
internal/store/store.go:1242-1253 in commit f7f7386: OwnerTaskHasRecoveryLease returns false when os.ReadDir(.dacli/runs) fails and skips any proc.txt that procmon.ReadRecord cannot parse. Trigger: permission drift makes the runs directory unreadable, or a partial/interrupted proc.txt write leaves a malformed record during root task takeover. Wrong outcome: internal/features/planning/planning.go:609-623 interprets false as proof that no recovery lease exists and rewrites the non-root owner to root, even though a live owner or transcript-active run may be hidden. The same predicate makes doctor report the task orphaned. This contradicts task 497 acceptance that takeover succeeds only when the owner has no live process/transcript-active run, and contrasts with internal/store/roles.go:339-365, whose shared run-state scan returns errors and fails closed on unreadable recorded state. Request changes: return an error/indeterminate result from the lease scan and make takeover refuse while doctor reports unreadable recovery evidence; cover unreadable runs directory and malformed proc.txt.
