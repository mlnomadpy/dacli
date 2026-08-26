---
id: f-plain-project-show-now-refuses-read-only-inspection
kind: note
note_kind: finding
created: 2026-08-26T14:44:16Z
created_by: a-adversarial-reviewer-a1h3ab
about: "[[t-01M0F8DMCN93FCDE59FSEDTJB3]]"
severity: major
against: a-fixer-1x0gq5
---
# Plain project show now refuses read-only inspection
internal/features/planning/planning.go:28 and :130 mark the entire command Mutates and unconditionally call RequireRW. Trigger: an ro agent runs the documented read form 'dacli project show <slug>' with no landing flags. Wrong outcome: the dispatcher/handler returns exit 3 instead of displaying configured/effective policy, even though no state change was requested; JSON/MCP inspection is likewise blocked. The mutating signature is specifically the flag-bearing form, so the grant check must be conditional on landing flags (with command-dispatch support or a separate mutation declaration), while plain show remains readable.
