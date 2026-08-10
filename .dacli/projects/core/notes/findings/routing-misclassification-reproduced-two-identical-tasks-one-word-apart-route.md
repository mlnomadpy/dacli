---
id: f-routing-misclassification-reproduced-two-identical-tasks-one-word-apart-route
kind: note
note_kind: finding
created: 2026-08-10T14:36:32Z
created_by: a-root
about: "[[318]]"
severity: moderate
origin: internal/features/teamops/teamops.go:611
---
# Routing misclassification reproduced: two identical tasks, one word apart, route to different kinds
Reproduced in a clean workspace with a roster of exactly two roles (impl/implementer, rev/reviewer). Task A: 'Write the tests the suite audit calls for' -> 'kind reviewer (title verb "audit")'. Task B: 'Write the unit tests the suite requires' -> 'kind implementer (default)'. Identical work, identical estimate; the only difference is an incidental noun. ROOT CAUSE at teamops.go:611: inferKind ranges over strings.Fields(title) and returns on the FIRST word found in kindVerbs, wherever it appears — so a noun usage ('the suite audit', 'the design doc', 'a research spike') hijacks the classification from the leading verb that states the actual intent. OBSERVED IN PRODUCTION during the 2026-08-10 audit wave: task 315 ('Write the missing and strengthened tests the suite audit calls for') routed to the role literally named 'reviewer', whose charter is 'never implements' — for a task whose entire content is writing code. I overrode it by hand at spawn time, which is exactly the operator habit team assign exists to replace. Second, separate defect filed as 319: among roles of the SAME kind, only model price and capacity rank them, so a Go-code audit picked prompt-auditor (sonnet, cap 8) over go-auditor (opus) — cheapest capable, and unable to do the work.
