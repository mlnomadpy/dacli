---
id: d-persist-terminal-outcome-and-claim-release-in-one-process-record-transition
kind: note
note_kind: decision
created: 2026-08-13T15:42:11Z
created_by: a-fixer-dt88p4
about: "[[422]]"
---
# Persist terminal outcome and claim release in one process-record transition
## Chose
Persist terminal outcome and claim release in one process-record transition
## Rejected
Treat retirement alone as claim release
## Because
spawn claim checks consume proc.txt and retirement lives in a separate agent file; one atomic proc-record rename gives agents and claim gates the same terminal classification and supports identity-proven crash recovery.
