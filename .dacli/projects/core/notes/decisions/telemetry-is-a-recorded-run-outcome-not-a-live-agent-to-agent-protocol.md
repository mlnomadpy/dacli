---
id: d-telemetry-is-a-recorded-run-outcome-not-a-live-agent-to-agent-protocol
kind: note
note_kind: decision
created: 2026-08-04T16:14:42Z
created_by: a-root
---
# Telemetry is a recorded run outcome, not a live agent-to-agent protocol
## Chose
Telemetry is a recorded run outcome, not a live agent-to-agent protocol
## Rejected
agents heartbeating to each other, or a message bus between live runs
## Because
The failures this session were all ONE-directional and all about the operator, not about agents talking: a run that produced nothing (task 200's agent), a run blocked from using dacli at all (task 264's agent), a merge path that could not merge (the integrator). In every case an agent knew, and the knowledge died in a transcript nobody opened. What was missing is a durable, queryable record of how each run ENDED and whether it was still moving — which dacli already half-computes (finalizeRun's 'no visible result', the dashboard's stalled/thinking/acting state machine) but only on paths the operator may never take. So the work is 268 (finalize on exit), 269 (a blocked channel that survives dacli being the broken thing), 270 (state in agents, not only the dashboard) and 271 (a wave rollup). Agent-to-agent liveness is deliberately NOT in scope: two agents in a wave do not need to know each other's heartbeat, they need to not duplicate each other's filings, which is task 274 and is a read-freshness problem rather than a telemetry one. A message bus would be a second coordination substrate competing with the workspace, which is the thing the workspace exists to be.
