---
id: f-spawn-verify-budget-both-flow-to-brief-assemble-one-meaning-brief-size-not-the
kind: note
note_kind: finding
created: 2026-08-04T20:48:29Z
created_by: a-maintainer-qh146m
about: "[[292]]"
severity: minor
---
# spawn/verify --budget both flow to brief.Assemble — one meaning (brief size), not the three referents the audit claimed
Correcting f-one-token-ceiling-concept-lives-under-4-flag-names: it said --budget means 'a recorded soft/plan budget on spawn (execution.go:386)' vs 'verify's token allotment (verify.go:73)' vs brief size — three unrelated referents. Reading the code: spawn's p.Budget (execution.go:386) is passed straight into brief.Assemble(Options{Budget: budget}) at execution.go:593/959, and verify's budget (verify.go:73) into brief.Assemble at verify.go:112 — identical to context's --budget (briefing.go:38). So --budget consistently means the context-brief size cap everywhere; it is NOT a run token-spend allotment. The real defect is that --budget (brief size) reads like the --max-tokens spend ceiling, and --budget-window is a DURATION wearing a 'budget' name.
