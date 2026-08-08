---
id: d-filed-loop-priority-ordering-task-103-as-the-single-highest-value-change
kind: note
note_kind: decision
created: 2026-07-23T12:11:45Z
created_by: a-waq3de2hcs
about: [[084]]
---
# Filed loop priority-ordering (task 103) as the single highest-value change
## Chose
Filed loop priority-ordering (task 103) as the single highest-value change
## Rejected
unfilled() over-broad placeholder match ('...'/'{{' via strings.Contains, gates.go:465); duplicate the already-filed 096 (governor state persistence), 101 (stale codebase map), or 102 (spawn-refused force-close)
## Because
The onboard/gates TODO markers are false positives (scanner code + already covered by 101), and the loudest defects were already filed as 096/101/102. The loop building by Seq instead of MoSCoW/critical-path (orchestration.go:294-297 + readyTasks) is an unfiled, high-value correctness bug that directly defeats the loop's stated charter, is grounded in concrete file:line evidence, and has a ready fix (reuse cmdNext's ordering, insight.go:209-218). The unfilled() '...' collision is real but modest and only bites project-section content, so lower value.
