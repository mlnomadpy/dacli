---
id: f-checkallacceptance-rewrites-the-whole-acceptance-section-destroying-prose-and
kind: note
note_kind: finding
created: 2026-08-10T17:44:03Z
created_by: a-go-auditor-d451f3
about: "[[t-01KZ93DW7P6BQ1HHDWY7MEH2KJ]]"
source_event: 01KZPBRFP9WPJ1ZJJCKRQB2QNE
---
# CheckAllAcceptance rewrites the whole Acceptance section, destroying prose and nested checkboxes on every close
store.go:314 does t.Doc.SetSection("Acceptance", mdstore.RenderCheckboxes(boxes)). RenderCheckboxes (mdstore.go:692-704) emits ONLY flattened '- [x] text' lines; Checkboxes (mdstore.go:677-689) extracts only lines whose trimmed form starts with '- [ ] '/'- [x] '. So on every close path (task done --all -> acceptance.go:181; accept -> acceptance.go:273) the Acceptance section is REPLACED by just its checkboxes: any prose line, blank line, plain '- bullet', or the indentation of a nested '  - [ ] sub-item' is permanently discarded on disk. The doc comment at store.go:296 claims it marks boxes done 'in place', which the code violates. Latent in this repo's dogfood tasks (all pure-checkbox today) but LIVE for any user whose Acceptance section carries explanatory text or nested criteria — the close silently mutates the record it is meant to only tick. Fix: flip [ ]->[x] on the existing checkbox lines while preserving every other line and its indentation, rather than re-rendering the section from Checkboxes().
