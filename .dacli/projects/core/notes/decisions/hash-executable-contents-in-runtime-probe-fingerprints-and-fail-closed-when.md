---
id: d-hash-executable-contents-in-runtime-probe-fingerprints-and-fail-closed-when
kind: note
note_kind: decision
created: 2026-08-12T15:27:36Z
created_by: a-codex-maintainer-zszvv9
about: "[[374]]"
---
# Hash executable contents in runtime probe fingerprints and fail closed when bytes cannot be read
## Chose
Hash executable contents in runtime probe fingerprints and fail closed when bytes cannot be read
## Rejected
Continue using size and modification time, or add another mutable metadata field
## Because
Metadata can be deliberately preserved during in-place replacement; a SHA-256 content identity binds the cached authorization verdict to the probed executable, and an unreadable executable cannot safely hydrate a verified verdict.
