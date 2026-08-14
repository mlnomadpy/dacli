---
id: 01KZYWZ2HSBNSJBRZK7PV0VJQ1
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-14T01:07:44Z
created_by: a-maintainer-0c0w8g
about: "[[t-01KZYRZPSGFQRPAWRYM6NACDCC]]"
origin: agent
applied: true
checksum: sha256:fbbc04c379a5fda39a6e535c178e01220bb76991e4991f84ff143cde9ebae3d1
---
aa25529 448: preserve effective landing policy across loop recovery

Resolve CLI > project > legacy landing policy in orchestration, persist it beside landing checkpoints, forward explicit overrides to ship, fail closed when PR mode has no remote, and document configuration/recovery.

Mutation proof: forcing configured.Mode = local failed TestLoopResolvesAndForwardsProjectLandingPolicy with effective policy = {Mode:local Base:main} override=false.
role: maintainer
