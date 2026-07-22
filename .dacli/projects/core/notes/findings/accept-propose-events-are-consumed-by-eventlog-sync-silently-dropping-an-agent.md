---
id: f-accept-propose-events-are-consumed-by-eventlog-sync-silently-dropping-an-agent
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-cw27djtx7d
about: [[t-01KY5GP5SE357058NVZKAG0WJ5]]
source_event: 01KY5GXWM33NA3VR8E7Y6Q5GDC
github:
  issue: 16
  repo: mlnomadpy/dacli
---
# accept-propose events are consumed by eventlog.Sync, silently dropping an agent's acceptance proposal
acceptance.propose (internal/features/acceptance/acceptance.go:80-84) records a read-only agent's acceptance proposal as a plain model.EventComment whose body carries the 'accept-propose:' prefix. accept --all later finds these via proposedTasks->eventlog.List(Kinds:[EventComment], Pending:true) filtered by isProposal (acceptance.go:186-208,211). BUT eventlog.Sync/apply treats EVERY EventComment identically (internal/eventlog/sync.go:155-160): it logOnce's the body to the task Log and MarkApplied's the event. Nothing distinguishes an accept-propose comment from a generic comment. So any Sync that runs between the agent's propose and the owner's 'accept --all' consumes the proposal: it becomes a Log line and applied:true, so proposedTasks (Pending:true) no longer sees it and accept --all reports 'no tasks proposed for acceptance' — the task is never closed, boxes never checked, with no signal. Sync runs on the documented owner path 'dacli sync' (collab.go:18,31, 'Apply pending child events to objects you own') AND automatically every supervise turn (execution.go:757). Fix: give accept-propose its own EventKind (or have eventlog.apply skip/deny comments whose body starts with the propose prefix) so the two consumers don't race on the same pending event.
