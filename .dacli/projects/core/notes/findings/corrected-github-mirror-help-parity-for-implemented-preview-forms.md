---
id: f-corrected-github-mirror-help-parity-for-implemented-preview-forms
kind: note
note_kind: finding
created: 2026-08-19T12:42:21Z
created_by: a-fixer-00g3ry
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
severity: major
---
# Corrected github mirror help parity for implemented preview forms
Confirmed in internal/features/ghmirror/ghmirror.go:47-48: github sync accepted project/task-window/push flags but advertised no arguments, while cmdPull at :1006 accepts --dry-run but omitted it from its Usage. internal/cli/usage_parity_invariant_test.go now preserves the intentionally delegated sync-to-pull missing-argument form with --dry-run.
