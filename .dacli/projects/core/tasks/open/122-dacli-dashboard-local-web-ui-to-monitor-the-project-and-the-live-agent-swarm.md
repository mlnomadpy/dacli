---
id: t-01KY8536HP81RV3XJNPC7WYVPV
kind: task
created: 2026-07-23T18:51:34Z
created_by: a-root
owner: a-root
priority: must
---
# dacli dashboard: local web UI to monitor the project and the live agent swarm
## Acceptance
- [ ] A new 'dacli dashboard' command serves a self-contained local web page showing projects, tasks by status/burndown, and the LIVE agents (swarm) with their task/role/runtime and last activity, read from the store + event log
- [ ] The page auto-refreshes (or streams) so a running loop's agents appear live; served on localhost with no external dependencies; covered by a handler test
## Log
