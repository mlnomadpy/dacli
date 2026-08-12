---
id: d-kept-runtime-doctor-read-classified-and-preserved-cooperative-best-effort
kind: note
note_kind: decision
created: 2026-08-12T13:21:50Z
created_by: a-codex-maintainer-3sbkdv
about: "[[365]]"
github:
  issue: 499
  repo: mlnomadpy/dacli
---
# Kept runtime doctor read-classified and preserved cooperative best-effort handling
## Chose
Kept runtime doctor read-classified and preserved cooperative best-effort handling
## Rejected
Mark doctor authority-mutating and drop every unverified sandbox argument under --cooperative
## Because
doctor writes only derived gitignored probe cache, while the mutating classification breaks the audited command capability table and trust-doc contract; preserving declaration-only args under the explicit cooperative escape keeps the existing runtime-argument round-trip contract, while arguments from an observed failed probe are safely omitted
