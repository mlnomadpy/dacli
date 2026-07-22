---
id: 01KY3ESJ5KACT5JDM2K6CQNW67
kind: event
event_kind: finding
created: 2026-07-21T23:04:52Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
origin: agent
applied: true
---
FindTask reads+parses the entire task tree per call; amplified to O(events×tasks) inside sync/taint/replay loops

store.FindTask (store.go:357) resolves ONE ref by calling ListTasks(w,"","") (store.go:287), which walks every project × every status folder and mdstore.ReadFile+Parses EVERY task .md, then linear-scans. It is then called inside loops: eventlog/sync.go:37 (once per pending event → O(events×tasks) full re-reads of the task tree per Sync); store/taint.go canonRef:116 (once per tainted hit → O(hits×tasks)); features/execution/replay.go:91 (once per run directory, with a loop-invariant taskRef → re-reads the whole tree per run dir). Fix: build the task list / an id|seq|slug→task index once, reuse it. Secondary: FindTask does two fmt.Sprintf allocs per task per call (store.go:365-366) — only needed when ref is numeric.
