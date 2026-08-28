---
id: 01M13TMEYPFRZPAZYBJX0729H4
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-28T09:18:53Z
created_by: a-root
about: "[[t-01M1068MTFPQ6YFVQG204M2EX4]]"
origin: agent
applied: true
checksum: sha256:b9ef1ff587c83435a83b6d3d983e37f1d124b000c3fb7fdcb85c0738906474a2
---
25102c96 t-01M1068MTFPQ6YFVQG204M2EX4: drain GitHub publisher process trees

Keep mutating gh calls attached to their complete process group across success, failure, interrupt, and timeout so the sequence lease is never released while descendants remain.

Mutation evidence: removing Setpgid made TestGitHubPushTimeoutKillsPublisherTreeBeforeReleasingLock hang and fail after 6s with the forked publisher retaining inherited streams.
role: root
