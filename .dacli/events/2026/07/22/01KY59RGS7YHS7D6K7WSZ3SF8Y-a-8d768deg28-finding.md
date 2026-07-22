---
id: 01KY59RGS7YHS7D6K7WSZ3SF8Y
kind: event
event_kind: finding
created: 2026-07-22T16:15:24Z
created_by: a-8d768deg28
about: [[t-01KY59FNF0CRTHD6SECSM2ZC6H]]
origin: agent
applied: true
---
decisions gate counts notes but never verifies a rejection, contradicting its own description

gates.go evaluate() case 'decisions' (gates.go:383-388) returns Desc '≥N decision(s) with a rejection' but the OK test is only 'len(notes) >= want' — it counts decision notes and never inspects whether any carries a Rejected section. A decision note with an empty/absent 'Rejected' passes the gate. This violates the package's stated 'FILLED, not present' contract (gates.go:8-10) and the design ethos that the rejected alternative is 'the valuable part' of a decision note. All shipped templates use 'decisions: min 1' (tpl/{standard,product,research-paper}.md), so every project's design gate can be cleared with rejection-free decisions. Fix: count only notes whose Section("Rejected") is non-empty (mirror brief.go:137).
