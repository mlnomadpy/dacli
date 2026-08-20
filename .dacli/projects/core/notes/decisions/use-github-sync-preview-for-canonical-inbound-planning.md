---
id: d-use-github-sync-preview-for-canonical-inbound-planning
kind: note
note_kind: decision
created: 2026-08-19T12:38:58Z
created_by: a-fixer-5cv5vk
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
github:
  issue: 730
  repo: mlnomadpy/dacli
---
# Use github sync preview for canonical inbound planning
## Chose
Use github sync preview for canonical inbound planning
## Rejected
Document github pull <project> --dry-run
## Because
The shipped pull Usage accepts only <project>, while github sync supports --dry-run and previews both halves.
