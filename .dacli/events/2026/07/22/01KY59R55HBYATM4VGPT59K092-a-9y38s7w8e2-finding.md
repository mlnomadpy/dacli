---
id: 01KY59R55HBYATM4VGPT59K092
kind: event
event_kind: finding
created: 2026-07-22T16:15:12Z
created_by: a-9y38s7w8e2
about: [[t-01KY59FNFK27A1084PQ8R2CJ5S]]
origin: agent
applied: true
---
collab threads mis-attributes answers: answered-by is keyed per-task, not per-question

collab.go:181-184 builds answered[e.About]=e.Actor keyed on the task ID (EventAnswer.About is t.ID, set in cmdAnswer collab.go:143), keeping only the FIRST answer seen per task. collab.go:190 then renders 'answered by '+answered[q.About] for every answered question about that task. If a task has two questions answered by different agents, both display the same (first) actor. The per-question OPEN/answered status itself is correct (uses q.Applied, collab.go:189), so this is attribution-only, but it silently prints the wrong answerer. Fix: key the answer actor by the answered question's id rather than by its task, e.g. carry the resolved question id on the EventAnswer (or match on MarkApplied'd question).
