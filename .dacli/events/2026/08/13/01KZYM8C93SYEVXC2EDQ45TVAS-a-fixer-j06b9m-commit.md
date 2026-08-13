---
id: 01KZYM8C93SYEVXC2EDQ45TVAS
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-13T22:35:32Z
created_by: a-fixer-j06b9m
about: "[[t-01KZYHZTP8NJVY9TYM7S2ANJ38]]"
origin: agent
applied: true
checksum: sha256:7716b1a9fdcd21f3c9ec9ab5ca14b3d65a25d904a98d1e7188771676114be878
---
0d28097 437: preserve distinct decision mirror payloads

Compare normalized Chose, Rejected, and Because sections before grouping near-duplicate decision titles so repeated decisions collapse without hiding materially different choices.

Mutation: removing the payload guard fails TestCanonicalNoteFilesKeepDecisionsWithDifferentPayloads with "materially different decisions collapsed to 1 record(s)".
role: fixer
