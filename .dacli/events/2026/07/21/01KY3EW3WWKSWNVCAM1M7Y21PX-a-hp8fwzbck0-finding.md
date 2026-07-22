---
id: 01KY3EW3WWKSWNVCAM1M7Y21PX
kind: event
event_kind: finding
created: 2026-07-21T23:06:16Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
origin: agent
applied: true
---
Two avoidable algorithmic inefficiencies in spm: kahn re-sorts the ready frontier every pop; maskCode copies the whole tail per code fence

criticalpath.go:217 — the sort.Slice(ready, ...) sits INSIDE the for-len(ready)>0 loop, so the frontier is fully re-sorted on each of V iterations, giving O(V^2 logV) instead of O(V logV); a container/heap keyed on pos, or an order-preserving insert, removes it. ambiguity.go:221 — string(b[i+3:]) copies the entire remaining buffer to a new string on each fenced-block scan; bytes.Index on the byte slice searches with zero allocation. Both minor (inputs are small) but clear-cut.
