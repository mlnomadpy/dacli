---
id: d-file-the-burn-ro-role-population-filter-class-fix-as-the-single-highest-value
kind: note
note_kind: decision
created: 2026-07-26T17:05:18Z
created_by: a-1hwz5pcjva
about: [[084]]
---
# file the burn ro-role population-filter class fix as the single highest-value change
## Chose
file the burn ro-role population-filter class fix as the single highest-value change
## Rejected
backfilling role_kind: reviewer into .dacli/roles/reviewer.md as the fix (what the original finding proposed), or filing the older major findings (calibrate by-agent-band empty, operator record-commit unguarded, taint under-broad) or the H7/kill-checkpoint design reframes
## Because
The reviewer.md finding is the freshest (2026-07-26), verifiable at file:line, and directly undermines just-merged code (task 149, commit 828f164) whose whole purpose is an accurate 1.5x overspend yell -- so it beats the older a-root findings (likely already triaged) and the two 'moderate' design reframes (H7 mid-run budget gauge, kill checkpoint signal) which are speculative feature ideas, not defects in shipped code. A pure data backfill fixes one file but not the class: any ro role authored without role_kind re-contaminates the Rate, and it is not unit-testable. Excluding grant==ro runs in burn.go reviewerRoles aligns the Rate population with the Ceiling population by the property that actually defines that population (an ro role cannot complete a task so is never a calibration sample), closes the class, is unit-testable, and mirrors the 'fix the class not the instance' pattern of decisions 084/116.
