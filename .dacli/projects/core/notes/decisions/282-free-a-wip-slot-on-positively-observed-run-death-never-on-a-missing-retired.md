---
id: d-282-free-a-wip-slot-on-positively-observed-run-death-never-on-a-missing-retired
kind: note
note_kind: decision
created: 2026-08-04T20:41:43Z
created_by: a-maintainer-4pjwbf
about: "[[282]]"
---
# 282: free a WIP slot on positively-observed run death, never on a missing 'retired' field
## Chose
282: free a WIP slot on positively-observed run death, never on a missing 'retired' field
## Rejected
eventlog 'recent activity' inside store.ActiveInRole
## Because
store is L2 and its own header forbids touching the event log (eventlog imports store, so the reverse is an import cycle). Run records (proc.txt) already give store both signals via its existing procmon dep: procmon.AliveRecord for liveness and Record.Started for recency. holdsWIPSlot frees a slot only when every run is provably dead AND the newest is past a 1h grace window; an agent with NO run record keeps its slot (absence of a run is not evidence of death). This puts the fix in the single WIP denominator (ActiveInRole) so the gate and doctor both benefit with no slice duplication.
