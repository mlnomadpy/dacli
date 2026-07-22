---
id: d-put-pr-first-integration-in-cmdintegrate-printegratetask-ship-just-forwards-pr
kind: note
note_kind: decision
created: 2026-07-22T21:42:05Z
created_by: a-c79p0msrw8
about: [[078]]
---
# Put PR-first integration in cmdIntegrate (prIntegrateTask); ship just forwards --pr/--no-merge/--merge
## Chose
Put PR-first integration in cmdIntegrate (prIntegrateTask); ship just forwards --pr/--no-merge/--merge
## Rejected
Duplicate the push+pr+gh-merge loop inside the ship slice, shelling dacli push/pr/gh per task
## Because
The per-task merge loop already lives in the vcs slice (mergeTask, cmdIntegrate) alongside push/pr/prBody/verdicts — implementing --pr there reuses openPR and mergeTask directly (same package, no cross-slice import) and keeps ONE integration code path; ship shells 'dacli integrate' already, so it only needs to pass the flags through. gh and push are injected via package vars (runGH, pushBranch) so the network path is unit-testable without a live GitHub.
