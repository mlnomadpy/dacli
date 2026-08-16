---
id: 01M05V3VYKFYWWYQAEV4J2ZA1K
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-16T17:50:05Z
created_by: a-maintainer-n5gm5y
about: "[[t-01KZZR4CN0HWN232ZD2GYGQDFP]]"
origin: agent
applied: true
checksum: sha256:ec5ba55077294de4a434ba6f424bbc05c89a379ff17ff33ce01011dee4d2d6b4
---
3ef7ba7 t-01KZZR4CN0HWN232ZD2GYGQDFP: prevent in-flight review duplicates

Compare candidate problem and acceptance content before task allocation, and place the completed wave identity, branch tip, linked issue, and pending landing state in the review anchor brief.

Mutation: raising both structured overlap thresholds above 1 made TestFindNearDuplicateTaskContentCatchesInFlightGeneratedRefDuplicate fail at similarity_test.go:106 with duplicate=nil.
role: maintainer
