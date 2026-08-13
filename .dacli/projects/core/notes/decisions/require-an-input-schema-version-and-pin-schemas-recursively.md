---
id: d-require-an-input-schema-version-and-pin-schemas-recursively
kind: note
note_kind: decision
created: 2026-08-13T22:00:14Z
created_by: a-fixer-sn8j7p
about: "[[435]]"
github:
  issue: 639
  repo: mlnomadpy/dacli
---
# Require an input schema version and pin schemas recursively
## Chose
Require an input schema version and pin schemas recursively
## Rejected
Version only the tools/list metadata or compare golden JSON byte-for-byte
## Because
A required schema_version lets the dispatcher reject unsupported calls before mutation, while recursive subset comparison preserves stable fields without rejecting additive descriptions or properties
