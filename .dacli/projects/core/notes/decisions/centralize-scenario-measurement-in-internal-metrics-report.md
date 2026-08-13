---
id: d-centralize-scenario-measurement-in-internal-metrics-report
kind: note
note_kind: decision
created: 2026-08-13T21:25:14Z
created_by: a-fixer-q7facv
about: "[[433]]"
github:
  issue: 631
  repo: mlnomadpy/dacli
---
# Centralize scenario measurement in internal/metrics Report
## Chose
Centralize scenario measurement in internal/metrics Report
## Rejected
Calculate figures separately in insight JSON, human output, and wscore
## Because
one collector preserves identical named-window scoping, sample counts, failure classes, and null semantics across consumers without feature-slice imports
