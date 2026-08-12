---
id: d-infer-architectural-claim-roots-from-concrete-behavioral-vocabulary-in-the
kind: note
note_kind: decision
created: 2026-08-12T16:48:08Z
created_by: a-codex-maintainer-nmzkpw
about: "[[385]]"
---
# Infer architectural claim roots from concrete behavioral vocabulary in the whole task
## Chose
Infer architectural claim roots from concrete behavioral vocabulary in the whole task
## Rejected
Claim the entire repository whenever acceptance contains path-free behavior
## Because
Broad repository claims would serialize unrelated loop work; bounded signals such as runtime presets/persistence, adapters/doctor/sandbox, and CLI/runtime-add identify the required implementation slices while leaving unrelated paths refused.
