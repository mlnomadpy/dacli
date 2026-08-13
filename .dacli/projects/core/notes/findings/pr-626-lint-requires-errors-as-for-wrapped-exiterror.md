---
id: f-pr-626-lint-requires-errors-as-for-wrapped-exiterror
kind: note
note_kind: finding
created: 2026-08-13T21:42:15Z
created_by: a-root
about: "[[432]]"
severity: major
---
# PR 626 lint requires errors.As for wrapped ExitError
GitHub Actions run 31746101529 job 94601978696 failed at internal/store/verification.go:53:16 (errorlint): type assertion on error will fail on wrapped errors; use errors.As to check for *exec.ExitError. Required correction: import errors, replace direct assertion with errors.As, add/adjust a focused test if needed, commit through dacli, and push task 432.
