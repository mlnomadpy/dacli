---
id: d-v2-1-direction-make-the-orchestrator-loop-native-e-series
kind: note
note_kind: decision
created: 2026-07-22T14:29:41Z
created_by: a-root
---
# v2.1 direction: make the orchestrator loop native (E-series)
## Chose
v2.1 direction: make the orchestrator loop native (E-series)
## Rejected
keep the agent-done -> integrated gap as manual operator git+bookkeeping
## Because
this session proved async parallel agents work, but the operator is still hand-closing tasks, hand-staging, hand-merging, and hand-committing the record on every wave; the friction is no longer inside a spawn, it is the loop AROUND spawns — close it so a fleet runs with the operator setting policy, not doing bookkeeping
