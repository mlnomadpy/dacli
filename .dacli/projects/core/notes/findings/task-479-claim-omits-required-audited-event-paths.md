---
id: f-task-479-claim-omits-required-audited-event-paths
kind: note
note_kind: finding
created: 2026-08-20T08:34:48Z
created_by: a-maintainer-d7gr0n
about: "[[t-01M0CZANAQKP50AWEN2C6C8VXR]]"
severity: major
---
# Task 479 claim omits required audited-event paths
dacli commit refused verified task 479 changes because internal/eventlog/sync.go and internal/model/model.go are outside the current claim. Those paths are required to define the typed dependency event and materialize read-only proposals during owner sync. README.md can be dropped in favor of claimed docs, but the two Go paths cannot be avoided without losing acceptance behavior. No --force override was used.
