---
id: f-direct-integrate-and-accept-recovery-guidance-forms-a-policy-refusal-cycle
kind: note
note_kind: finding
created: 2026-08-28T13:42:14Z
created_by: a-maintainer-wg7dnx
about: "[[t-01M12QX9HEPKAAS1033W6HS45D]]"
severity: major
---
# Direct integrate and accept recovery guidance forms a policy-refusal cycle
internal/features/vcs/lifecycle.go integrationTasks refuses an open explicit task and recommends dacli accept; internal/features/acceptance/landed.go unlandedRefusal refuses that accept and recommends merging. internal/features/ship/ship.go accepts only pending proposals via accept --all, then explicitWave requires done, so an explicitly selected fully checked open task without a proposal cannot enter ship.
