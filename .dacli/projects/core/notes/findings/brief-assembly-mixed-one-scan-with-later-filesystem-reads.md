---
id: f-brief-assembly-mixed-one-scan-with-later-filesystem-reads
kind: note
note_kind: finding
created: 2026-08-20T08:08:13Z
created_by: a-maintainer-p5kmb7
about: "[[t-01M0AEG5K7JF96HV0RJ5K17NJN]]"
severity: major
---
# Brief assembly mixed one scan with later filesystem reads
internal/brief/brief.go previously hoisted tasks and events but still loaded decisions, findings, risks, glossary, lessons, roles, shortcuts, and calibration during rendering; LoadView now establishes one explicit boundary and AssembleView is filesystem-free.
