---
id: 01KZ6RX7907TWN8RWFZ6N8EAQT
kind: event
event_kind: commit
created: 2026-08-04T16:15:03Z
created_by: a-root
origin: agent
applied: true
---
5f081a1 file the telemetry and robustness backlog (268-274)

Every failure this session that cost a run was silence, not an error.

268 — a detached run is never finalized. finalizeRun already computes
'no visible result' for a child that left no events and no checked
acceptance, but it only runs on the `dacli wait` path. I spawned
detached and polled `agents` all day, so I never saw it: the run that
produced nothing still reads 'outcome: running (detached)' hours after
the process exited.

269 — an agent blocked from running dacli cannot report it, because
escalate, ask and note add all route through the binary that is broken.
Task 264's auditor hit exactly this, put the failure in its final
message, and it reached me only because I read a transcript.

270 — `agents` prints RAM and uptime, not whether an agent is working
or stuck. The dashboard already has the state machine
(thinking/acting/waiting/stalled); the CLI does not.

271 — no wave rollup, so reading a wave means opening six transcripts.

272 — one preflight for role/runtime capability. Three separate
instances landed today (250, 267, and the integrator's gh-vs-runtime
prompt), each fixed with its own check and no general gate.

273 — accept moves the task file and leaves the move uncommitted; three
occurrences today makes it a design gap, not carelessness.

274 — a brief is frozen at spawn, so agents in a wave cannot see each
other's filings.

The scoping decision is recorded: telemetry here means a recorded run
outcome, not a live agent-to-agent protocol. A message bus would be a
second coordination substrate competing with the workspace, which is
the thing the workspace already is.
role: root
