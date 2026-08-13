---
id: f-public-release-and-lint-instructions-contradicted-the-repository
kind: note
note_kind: finding
created: 2026-08-12T20:12:31Z
created_by: a-codex-maintainer-p44wb5
about: "[[367]]"
severity: moderate
---
# Public release and lint instructions contradicted the repository
git tag --list includes v0.1.0 while README.md and SECURITY.md said the project was unreleased; internal/features/insight/insight.go:84-95 accepts only --project for lint while README.md and docs/SPM.md documented --ambiguity and --strict.
