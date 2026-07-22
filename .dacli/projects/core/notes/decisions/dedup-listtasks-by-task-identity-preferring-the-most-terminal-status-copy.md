---
id: d-dedup-listtasks-by-task-identity-preferring-the-most-terminal-status-copy
kind: note
note_kind: decision
created: 2026-07-22T15:34:27Z
created_by: a-7zg8j1n976
about: [[038]]
---
# Dedup ListTasks by task identity, preferring the most-terminal status copy
## Chose
Dedup ListTasks by task identity, preferring the most-terminal status copy
## Rejected
Make FindTask tolerate two hits for the same id, leaving ListTasks yielding duplicates
## Because
The duplicate is a data-integrity DRIFT, not a normal state. Fixing it at ListTasks (the one walker every reader funnels through) means FindTask, TaskIndex, doctor, standup — every consumer — sees one task, not just FindTask. Preference = statusRank via AllStatuses order (done>blocked>active>open), tie-broken by mtime, so a stale open copy never shadows the authoritative done copy. The raw duplicated view stays reachable via DuplicateTaskFiles for the doctor check that must still SEE the drift.
