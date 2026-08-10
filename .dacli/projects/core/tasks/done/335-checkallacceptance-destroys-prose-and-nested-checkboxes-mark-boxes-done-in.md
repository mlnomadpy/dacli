---
id: t-01KZPBRQYVYY0F9YWVYK9AS4W3
kind: task
created: 2026-08-10T17:33:16Z
created_by: a-go-auditor-d451f3
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# CheckAllAcceptance destroys prose and nested checkboxes: mark boxes done in place, do not rewrite the Acceptance section
## Acceptance
- [x] CheckAllAcceptance flips '- [ ]' to '- [x]' on the existing checkbox lines while preserving every other line in the Acceptance section verbatim (prose lines, blank lines, plain '- bullets') and the leading indentation of nested checkboxes
- [x] a store-level test builds a task whose Acceptance section has a leading prose line, a nested '  - [ ] sub-item', and a trailing prose line, calls CheckAllAcceptance, SaveTask, re-reads from disk, and asserts every non-checkbox line and the nested indentation survive while all boxes read [x]
- [x] the test fails against the current SetSection(RenderCheckboxes(...)) implementation and passes after the fix (red-green shown)
- [x] the returned newly-checked count is unchanged and existing callers (acceptance.go:181, acceptance.go:273) and their tests stay green
## Log
- 2026-08-10T17:37:28Z claimed by a-junior-3jkefk
- 2026-08-10T17:44:03Z adopted by a-root (owner a-go-auditor-d451f3 orphaned)
- 2026-08-10T17:44:03Z accepted by a-root (applied 1 proposal(s))
- 2026-08-10T17:44:03Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T17:44:03Z completed by a-root
- 2026-08-10T17:44:04Z deliverable: dacli/335-checkallacceptance-destroys-prose-and-nested-checkboxes-mark-boxes-done-in exists but is NOT in trunk — closed anyway
- 2026-08-10T17:52:23Z accepted by a-root
- 2026-08-10T17:52:23Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T17:52:23Z deliverable: no dacli/335-checkallacceptance-destroys-prose-and-nested-checkboxes-mark-boxes-done-in branch — nothing to check against trunk
- 2026-08-10T17:52:23Z completed by a-root
