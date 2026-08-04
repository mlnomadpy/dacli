---
id: f-kind-aware-routing-works-but-is-opt-in-so-the-default-answer-is-still-the
kind: note
note_kind: finding
created: 2026-08-04T13:06:02Z
created_by: a-root
origin: internal/team/routing.go
---
# Kind-aware routing works but is opt-in, so the default answer is still the cheapest role of any kind
Measured on task 264 ('Audit the code that changed today and file what is worth fixing', Te 2.3), immediately after task 200 declared max_points across the roster:

  dacli team assign 264                    -> junior
  dacli team assign 264 --kind reviewer    -> prompt-auditor
  dacli team assign 264 --kind implementer -> junior

So the kind filter that task 238 added is real and it works. The problem is that nothing supplies it. With --kind absent, capacity is the only filter, and an audit that costs 2.3 points fits junior's cap of 3 — so the cheapest role of ANY kind wins, and an auditing task routes to an implementer on the cheap model.

The caps landed today fixed the other half: before them every task routed to junior regardless of size, because 15 of 18 roles declared no cap at all. Now size is respected. Kind is not, unless the caller already knows to ask — which means the operator has to have made the routing decision before consulting the thing that exists to make it.

Filed as 265. I overrode to go-auditor for this run.
