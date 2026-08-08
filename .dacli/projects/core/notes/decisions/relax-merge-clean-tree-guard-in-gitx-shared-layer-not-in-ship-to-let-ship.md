---
id: d-relax-merge-clean-tree-guard-in-gitx-shared-layer-not-in-ship-to-let-ship
kind: note
note_kind: decision
created: 2026-07-22T16:28:52Z
created_by: a-jjwx3z556n
about: [[050]]
---
# Relax merge clean-tree guard in gitx (shared layer), not in ship, to let ship tolerate a dirty .dacli
## Chose
Relax merge clean-tree guard in gitx (shared layer), not in ship, to let ship tolerate a dirty .dacli
## Rejected
Reorder ship to commit .dacli before integrate
## Because
The half-ship root cause is that dacli's normal state dirties .dacli (accept rename-moves tracked task files) yet gitx.Merge's IsClean guard refuses ANY dirty tracked file. Fixing it in gitx via IsCleanExcept('.dacli') makes BOTH manual 'dacli integrate/merge' and 'dacli ship' tolerate dacli's own workspace churn while still refusing dirty CODE. Reordering ship's commits would fix only ship and leave manual integrate broken, and would commit a premature 'after integrating' record.
