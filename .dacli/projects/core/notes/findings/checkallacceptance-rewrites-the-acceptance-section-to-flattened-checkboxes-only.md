---
id: f-checkallacceptance-rewrites-the-acceptance-section-to-flattened-checkboxes-only
kind: note
note_kind: finding
created: 2026-08-10T15:20:12Z
created_by: a-go-auditor-qz3zb9
about: "[[t-01KZ93DW7P6BQ1HHDWY7MEH2KJ]]"
source_event: 01KZP4377ECBTZY428YTEYNP91
---
# CheckAllAcceptance rewrites the Acceptance section to flattened checkboxes only, dropping any other content
store.CheckAllAcceptance (internal/store/store.go:301-315) reads the Acceptance section, extracts ONLY '- [ ]/- [x]' lines via mdstore.Checkboxes, then SetSection('Acceptance', RenderCheckboxes(boxes)) — replacing the entire section with re-rendered top-level checkbox lines. mdstore.RenderCheckboxes (mdstore.go:692-704) always writes at column 0 with no surrounding content. So on EVERY owner close (accept/task done/propose-sync all call this), any non-checkbox content in the Acceptance section is permanently deleted from the persisted task record: prose clarifications, blank lines, and — because Checkboxes ignores indentation — nested/indented sub-checkboxes are flattened to top level. Latent today (a scan of core task files found no acceptance section with non-checkbox or indented content), but this repo hand-edits task markdown, so the first annotated acceptance criterion will be silently lost on close. Recorded as a lead for a later cycle; not this cycle's top task because it currently loses nothing.
