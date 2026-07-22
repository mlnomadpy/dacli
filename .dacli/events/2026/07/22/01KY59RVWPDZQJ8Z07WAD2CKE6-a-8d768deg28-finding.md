---
id: 01KY59RVWPDZQJ8Z07WAD2CKE6
kind: event
event_kind: finding
created: 2026-07-22T16:15:35Z
created_by: a-8d768deg28
about: [[t-01KY59FNF0CRTHD6SECSM2ZC6H]]
origin: agent
applied: true
---
critical-path includes blocked tasks as schedulable while next excludes them

Inconsistent treatment of blocked tasks between two CPM readouts over the same graph. cmdCriticalPath builds open[] as everything not done (insight.go:422 bare 'else'), so blocked tasks become CPM nodes and appear in the schedule/critical path as if runnable. cmdNext excludes blocked from open[] (insight.go:138). So 'dacli critical-path' can star a blocked task as the thing to spawn children on first, contradicting 'dacli next'. Pick one policy: either both exclude blocked, or both include them and mark blocked in output.
