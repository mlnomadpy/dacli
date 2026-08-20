---
id: f-corrected-unsupported-github-pull-preview-form-in-canonical-docs
kind: note
note_kind: finding
created: 2026-08-19T12:38:58Z
created_by: a-fixer-5cv5vk
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: major
---
# Corrected unsupported github pull preview form in canonical docs
Fresh help on 2026-08-19 reports github pull usage as 'dacli github pull <project>' with no --dry-run. docs/OPERATOR_PLAYBOOK.md:19, skills/dacli/references/critical-path-github.md:3, and skills/dacli/references/github-landing.md:53 now use supported github sync <project> --dry-run for a bidirectional preview; docs/support_claims_test.go rejects the stale pull form.
