---
id: d-118-default-adopt-s-slug-to-the-sole-existing-project-or-refuse-if-2-instead-of
kind: note
note_kind: decision
created: 2026-07-26T23:13:08Z
created_by: a-yms7m1bbzj
about: [[118]]
---
# 118: default adopt's slug to the sole existing project (or refuse if 2+) instead of a dirname-derived guess; added project rm --force for recovery
## Chose
118: default adopt's slug to the sole existing project (or refuse if 2+) instead of a dirname-derived guess; added project rm --force for recovery
## Rejected
Only adding project rm and leaving the dirname-guess default alone
## Because
the dirname guess is the root cause (dirname 'dacli' != project 'core'), not just the missing recovery path -- fixing only the symptom (no rm command) would let the same silent-second-project bug recur on every future adopt; the fix needed both: correct the default AND add a recovery command for whoever hits the ambiguous (2+ project) refusal or an old mistake
