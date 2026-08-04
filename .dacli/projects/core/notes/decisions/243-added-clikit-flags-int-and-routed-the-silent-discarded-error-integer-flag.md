---
id: d-243-added-clikit-flags-int-and-routed-the-silent-discarded-error-integer-flag
kind: note
note_kind: decision
created: 2026-08-04T10:20:02Z
created_by: a-maintainer-0a7fqr
about: "[[243]]"
---
# 243: added clikit.Flags.Int and routed the silent discarded-error integer-flag sites through it
## Chose
243: added clikit.Flags.Int and routed the silent discarded-error integer-flag sites through it
## Rejected
converting the already-loud flag parses (planning --n range check, dashboard --port, parseLimit) and the non-Flags mcp/tools.go args-map int()
## Because
the acceptance is scoped to discarded-error sites; the three already-checked sites return tailored errors (range/HTTP-400) that a generic Int would regress, and mcp/tools.go i() reads an MCP args map not a *Flags so Flags.Int does not apply. Internal file-record parses (store/procmon/state/ghmirror/gates) are not user flags and stay as-is.
