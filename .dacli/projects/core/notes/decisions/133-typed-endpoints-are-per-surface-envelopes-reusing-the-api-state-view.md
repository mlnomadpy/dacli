---
id: d-133-typed-endpoints-are-per-surface-envelopes-reusing-the-api-state-view
kind: note
note_kind: decision
created: 2026-07-24T00:54:19Z
created_by: a-wey686j4cx
about: [[133]]
---
# 133: typed endpoints are per-surface envelopes reusing the /api/state view builders; /api/state kept
## Chose
133: typed endpoints are per-surface envelopes reusing the /api/state view builders; /api/state kept
## Rejected
replace /api/state with the four endpoints, or return bare arrays without a generated stamp
## Because
the SPA migration is a separate task and the legacy static/index.html still polls /api/state, so removing it would break the shipped dashboard (acceptance: existing behavior preserved). Each endpoint reuses buildProjectView/buildAgentView/liveAgents so the typed contract can never drift from the combined snapshot; an envelope with its own generated stamp lets a surface be polled independently and still reason about freshness.
