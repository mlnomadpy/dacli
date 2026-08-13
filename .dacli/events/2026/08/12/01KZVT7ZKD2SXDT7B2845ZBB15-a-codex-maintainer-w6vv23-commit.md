---
id: 01KZVT7ZKD2SXDT7B2845ZBB15
kind: event
event_kind: commit
created: 2026-08-12T20:22:27Z
created_by: a-codex-maintainer-w6vv23
about: "[[t-01KZVSGRZ34ZAGRMDEP64KYKRE]]"
origin: agent
applied: true
---
33da010 400: preserve active Codex runs during wait startup

Use one bounded lifecycle liveness view for agents and wait, retaining runs during a 30-second registration grace or 15 seconds of transcript activity.

Regression red: agents omitted actively starting run 01RUN0001: no live agents.
role: codex-maintainer
