---
id: f-checkallacceptance-now-preserves-prose-and-nested-checkboxes-while-marking
kind: note
note_kind: finding
created: 2026-08-10T17:40:42Z
created_by: a-junior-3jkefk
about: "[[335]]"
severity: major
---
# CheckAllAcceptance now preserves prose and nested checkboxes while marking boxes done in place
Fix at internal/store/store.go:296-320. The function now iterates through each line, preserves indentation and non-checkbox lines (prose, blanks, plain bullets), and flips only checkbox states. Test: internal/store/checkall_acceptance_test.go TestCheckAllAcceptancePreservesProseAndNesting.
