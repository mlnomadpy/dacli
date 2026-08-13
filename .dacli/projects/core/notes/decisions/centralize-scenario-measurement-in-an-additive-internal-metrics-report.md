---
id: d-centralize-scenario-measurement-in-an-additive-internal-metrics-report
kind: note
note_kind: decision
created: 2026-08-13T21:19:31Z
created_by: a-fixer-fwr9f3
about: "[[433]]"
github:
  issue: 630
  repo: mlnomadpy/dacli
---
# Centralize scenario measurement in an additive internal metrics report
## Chose
Centralize scenario measurement in an additive internal metrics report
## Rejected
Teach the human and JSON renderers to calculate their own figures
## Because
one lower-layer collector keeps window scoping, sample counts, failure classification, and missing-data semantics identical without a forbidden feature-slice import
