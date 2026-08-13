---
id: d-precompute-and-deconflict-the-planned-wave-claims
kind: note
note_kind: decision
created: 2026-08-13T14:58:53Z
created_by: a-fixer-dd0fvf
about: "[[419]]"
---
# Precompute and deconflict the planned wave claims
## Chose
Precompute and deconflict the planned wave claims
## Rejected
Rely only on the live spawn claim-overlap gate
## Because
The live gate refuses only after launching an earlier task and classifies the later task like retryable spawn failure; one precomputed claim map keeps dry-run/live derivation identical and lets unrelated tasks continue.
