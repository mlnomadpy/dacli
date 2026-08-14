---
id: 01KZYSFNP2QC2JRXADC5TG0Y35
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-14T00:06:54Z
created_by: a-maintainer-ry3evs
about: "[[t-01KZYRZPYGECMPYSEPDFS1QS3F]]"
origin: agent
applied: true
checksum: sha256:6dad4c9a77e842955d2bd721d3b211f1d66adfd1a7d11837eec9d073ce5bc135
---
15904a3 450: add typed project landing policy resolution

Persist validated local/pr policy fields and centralize CLI/config/default precedence so downstream commands consume one effective value. Project inspection now exposes only configured/effective policy metadata in JSON.

Mutation proof: changing the legacy default to pr failed TestResolveLandingPrecedence/legacy_default with got mode pr, want local.
role: maintainer
