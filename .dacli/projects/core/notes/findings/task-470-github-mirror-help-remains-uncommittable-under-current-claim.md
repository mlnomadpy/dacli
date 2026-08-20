---
id: f-task-470-github-mirror-help-remains-uncommittable-under-current-claim
kind: note
note_kind: finding
created: 2026-08-19T12:39:41Z
created_by: a-fixer-4rpd0f
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
severity: major
---
# Task 470 GitHub mirror help remains uncommittable under current claim
Reproduced with a red TestPullAndSyncUsageAdvertiseImplementedForms: internal/features/ghmirror/ghmirror.go:47 advertises dacli github sync with no project/window/dry-run form and :48 omits pull --dry-run; cmdPull at :1006 accepts --dry-run and cmdSync at :1149 forwards project, task refs, push window, --findings-as-issues, --with-tasks, and --dry-run. The required implementation also updates internal/cli/usage_parity_invariant_test.go. dacli commit refused those two files because this agent claim contains only internal/features/ghmirror/syncflags_test.go. Reverted the partial test and implementation; a follow-up must claim all three paths.
