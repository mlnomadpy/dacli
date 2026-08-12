---
id: 01KZV5MSY36WPM2ACA175JBN3D
kind: event
event_kind: commit
created: 2026-08-12T14:22:27Z
created_by: a-root
about: "[[t-01KZV4SAVNZ1JXMJ61RRQCJYPY]]"
origin: agent
applied: true
---
583ffc5 375: reproduce lost descendants without weakening PGID safety

Mutation red: TestRunStillLivePreservesTask177AfterLeaderExit failed with reconciliation lost a genuine helper after its recorded leader exited. Task-369 liveness and kill safety controls remained green.
role: root
