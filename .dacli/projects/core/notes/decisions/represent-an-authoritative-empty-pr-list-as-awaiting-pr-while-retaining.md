---
id: d-represent-an-authoritative-empty-pr-list-as-awaiting-pr-while-retaining
kind: note
note_kind: decision
created: 2026-08-12T20:10:23Z
created_by: a-codex-maintainer-csf6ta
about: "[[399]]"
github:
  issue: 540
  repo: mlnomadpy/dacli
---
# Represent an authoritative empty PR list as awaiting-pr while retaining ancestry-only merge detection
## Chose
Represent an authoritative empty PR list as awaiting-pr while retaining ancestry-only merge detection
## Rejected
Treat every unmerged branch with no PR as closed-unmerged orphaned
## Because
Only an explicit CLOSED PR proves the safe-retry case; an empty list means PR creation is still due and must preserve the commit refs.
