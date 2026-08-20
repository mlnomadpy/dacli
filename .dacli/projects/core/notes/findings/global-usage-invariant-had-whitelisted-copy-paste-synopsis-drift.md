---
id: f-global-usage-invariant-had-whitelisted-copy-paste-synopsis-drift
kind: note
note_kind: finding
created: 2026-08-19T12:32:04Z
created_by: a-fixer-3pqnc4
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
severity: major
---
# Global usage invariant had whitelisted copy-paste synopsis drift
internal/cli/usage_parity_invariant_test.go:26 and :30 listed handler forms for shortcut add and template show, so their unrelated table Usage strings were accepted. Adding a command-path prefix assertion reproduced failures for next, shortcut add, and template show before aligning internal/features/insight/insight.go:28, internal/features/shortcuts/shortcuts.go:24, and internal/features/stagegate/stagegate.go:17.
