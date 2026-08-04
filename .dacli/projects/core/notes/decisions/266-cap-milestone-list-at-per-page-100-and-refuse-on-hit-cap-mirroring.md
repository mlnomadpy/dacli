---
id: d-266-cap-milestone-list-at-per-page-100-and-refuse-on-hit-cap-mirroring
kind: note
note_kind: decision
created: 2026-08-04T14:42:12Z
created_by: a-maintainer-eswwm8
about: "[[266]]"
---
# 266: cap milestone list at per_page=100 and refuse on hit-cap, mirroring fetchAllIssues
## Chose
266: cap milestone list at per_page=100 and refuse on hit-cap, mirroring fetchAllIssues
## Rejected
using gh api paginate flag to read all pages unbounded
## Because
the paginate flag gives no truncation signal, so a caller could never tell a complete list from a partial one, which is the exact false-absence that duplicates milestones. The file convention (fetchAllIssues, task 205) is cap plus detect-cap plus refuse; per_page tops out at 100 on the milestones endpoint, so 100 is both page size and cap, and a hit-cap page missing the title errors rather than reporting a false absent.
