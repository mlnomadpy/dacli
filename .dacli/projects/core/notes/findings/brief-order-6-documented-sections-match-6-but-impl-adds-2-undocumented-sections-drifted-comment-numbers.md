---
id: f-brief-order-6-documented-sections-match-6-but-impl-adds-2-undocumented-sections-drifted-comment-numbers
kind: note
note_kind: finding
created: 2026-07-21T15:10:52Z
created_by: a-z4w30d506r
about: [[t-01KY2K6YF6PQ8CNKG0WM8WGFBS]]
---
# Brief order: 6 documented sections match §6, but impl adds 2 undocumented sections + drifted comment numbers
Compared internal/brief/brief.go Assemble() against ARCHITECTURE.md §6 canonical example (lines 113-149).

MATCH (relative order intact) for every section the doc shows: Task (brief.go:67) -> Why (78) -> Out of scope (83) -> Constraints (113) -> Risks (138) -> Glossary (148) -> What siblings found (199) -> Shortcuts (224). The doc's stated contract 'sections in fixed priority order, trim from bottom, task never trimmed' is honored: only Task is non-droppable (67, false arg); all others pass droppable=true; trim() walks bottom-up (brief.go:300).

MISMATCH — the doc's §6 example omits two sections the impl emits:
1. 'Lessons from other projects' inserted between Glossary and What siblings found (brief.go:168). Added by P1 (PROPOSALS), but ARCHITECTURE §6 was never updated, so the canonical example silently omits it.
2. 'Recent activity' inserted between What siblings found and Shortcuts (brief.go:209). Also absent from the §6 example.

CODE-COMMENT DRIFT (same silent-drift risk, internal): step comments are now inconsistent — Glossary is '// 6.' (brief.go:141) and What siblings found is ALSO '// 6.' (brief.go:171); Lessons is labeled '// 5b.' (brief.go:152) but physically sits AFTER step 6 (Glossary). The comment numbers no longer track emission order.

Recommendation: either add Lessons + Recent activity to the ARCHITECTURE §6 example (and renumber brief.go step comments 1-10 to match true order), or move them if the doc order is authoritative.
