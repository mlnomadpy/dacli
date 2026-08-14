---
id: 01KZYZH3P3Y56WQ5TP1CHYPK6Y
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-14T01:52:32Z
created_by: a-maintainer-11wpsr
about: "[[t-01KZYREKZGV6Z0CG3GM5J4G5BG]]"
origin: agent
applied: true
checksum: sha256:499f49861952117be22edab36a1c50b22a082dfdeaca32110a65021500c48607
---
00e9ab4 446: import GitHub acceptance checklists on pull

Move documented issue checklist boxes into canonical task Acceptance while preserving checked state and removing duplicate Context boxes. Bodies without a recognized checklist remain unchanged.

Mutation proof: discarding extracted acceptance fails TestPullImportsAcceptanceChecklistOnce with "canonical Acceptance has 0 boxes, want 2".
role: maintainer
