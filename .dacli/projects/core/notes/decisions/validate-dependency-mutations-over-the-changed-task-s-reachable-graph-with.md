---
id: d-validate-dependency-mutations-over-the-changed-task-s-reachable-graph-with
kind: note
note_kind: decision
created: 2026-08-28T00:18:36Z
created_by: a-maintainer-z0nyk9
about: "[[t-01M1068M8HJ9G8XCXMEMVE2V8D]]"
---
# Validate dependency mutations over the changed task's reachable graph with owning-project-first resolution
## Chose
Validate dependency mutations over the changed task's reachable graph with owning-project-first resolution
## Rejected
Validate the complete workspace graph on every dependency edit
## Because
Unrelated legacy faults must remain visible to readiness/inspection but cannot make all future graph mutations impossible; the reachable component is sufficient for missing-target and cycle safety of the requested edge.
