---
id: f-final-task-475-clarification-canonical-merge-state-and-runtime-cooldown
kind: note
note_kind: finding
created: 2026-08-19T12:31:03Z
created_by: a-root
about: "[[475]]"
severity: moderate
---
# Final task 475 clarification: canonical merge state and runtime cooldown visibility
Before owner acceptance, make the direct PR sequence explicit: pr or trusted --auto, wait until pr status reports merged and confirm trunk contains the commit, then owner accept and github push to close the issue. State separately that ship owns the governed accept-plus-integrate transaction for a reviewed wave. Also state that runtime doctor and run records expose health, but the shipped CLI has no dedicated cooldown-clear or expiry-inspection command; operators must not invent one.
