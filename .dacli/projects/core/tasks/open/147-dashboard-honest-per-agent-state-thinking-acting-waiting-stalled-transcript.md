---
id: t-01KY9W5QY0656EYEXRF70B2VY7
kind: task
created: 2026-07-24T10:54:09Z
created_by: a-root
owner: a-root
priority: must
---
# Dashboard: honest per-agent state (thinking|acting|waiting|stalled) + transcript/diff link (research shortlist #2, RICE 2.4)
## So that
the operator's daily thinking-vs-hung ambiguity and the adopter's presence-vs-artifact blindness are both killed
## Acceptance
- [ ] /api/agents derives an honest per-agent state (thinking|acting|waiting|stalled) from the transcript (which already contains it) + exposes a transcript/diff link per run; handler test
- [ ] The Vue AgentSwarm shows the state as a badge and a 'view transcript / see the diff' link; read-only; component test
## Log
