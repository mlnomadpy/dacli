---
id: f-docs-readme-md-status-column-verified-against-live-command-registration
kind: note
note_kind: finding
created: 2026-07-23T22:59:16Z
created_by: a-vwxfvnxmzb
about: [[129]]
severity: minor
---
# docs/README.md status column verified against live command registration
Verified all 4 acceptance items against source: shortcuts.go:24 registers 'shortcut promote' -> cmdPromote (shortcuts.go:65); skillforge.go:27 registers 'skill promote' -> cmdPromote (skillforge.go:71); ghmirror.go:420/489 define cmdPull/cmdSync (both registered, no planned() gate); grep -rn 'clikit.Planned(' across repo returns zero call sites (only the func def at clikit.go:76). Removed '(promote planned)' from SHORTCUTS.md and SKILLS.md rows, 'inbound planned' from GITHUB.md row, and the false 'still genuinely unimplemented planned() stubs' prose paragraph. Docs-only diff (docs/README.md, 4 lines changed), no Go source touched.
