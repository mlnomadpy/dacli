---
id: d-116-mechanical-near-duplicate-refusal-at-task-add-time-not-full-backlog-brief
kind: note
note_kind: decision
created: 2026-07-23T22:49:21Z
created_by: a-wgghcfe1sf
about: [[116]]
---
# 116: mechanical near-duplicate refusal at task-add time, not full-backlog brief injection
## Chose
116: mechanical near-duplicate refusal at task-add time, not full-backlog brief injection
## Rejected
passing the full open+recently-done task title list into the review auditor's spawn brief (suggested direction 1)
## Because
the auditor's brief is a static Context string set once when the standing improvement task is created (orchestration.go ensureImproveTask) -- refreshing it every cycle with a live backlog listing means editing shared context-assembly plumbing used by every spawn, not just the review auditor, and still only nudges an LLM rather than guaranteeing dedup. store.FindNearDuplicateTask + a refusal (exit 3) in cmdTaskAdd (planning.go) is a mechanical backstop that works even if the auditor ignores its brief, mirrors the existing spawn/accept --force refusal shape, and is fully unit-testable. Also updated ensureImproveTask's Context to tell the auditor to check the open backlog first (lightweight version of direction 1) as a cheap complement.
