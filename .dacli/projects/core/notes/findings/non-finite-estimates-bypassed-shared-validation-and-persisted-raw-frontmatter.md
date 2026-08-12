---
id: f-non-finite-estimates-bypassed-shared-validation-and-persisted-raw-frontmatter
kind: note
note_kind: finding
created: 2026-08-12T16:58:26Z
created_by: a-codex-maintainer-gqkrc4
about: "[[381]]"
severity: major
---
# Non-finite estimates bypassed shared validation and persisted raw frontmatter
Red regressions showed spm.ThreePoint.Valid accepted NaN in all three points and +Inf as pessimistic (internal/spm/spm_test.go), while task add and task estimate both returned success for Inf,Inf,Inf and wrote task files (internal/features/planning/planning_test.go and estimate_test.go).
