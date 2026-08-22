---
id: 01M0N22SCY9DDPQJY426N2G5Z3
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-22T15:40:26Z
created_by: a-root
about: "[[t-01M0AK4XK4M7CTJ6DXRKFW8XWG]]"
origin: agent
applied: true
checksum: sha256:f705a6a137a4e13ee2b3aaf0f7074c585029493b5fcee98c8e7e14ee4e973f88
---
d9f7e57 task 473: separate live WIP from identity history

Use one liveness-probed store census for spawn gates, team, and dashboard while preserving conservative minted identity provenance for role removal.

Mutation proof: replacing live-run iteration with identity iteration made TestRoleWiredIntoSpawn fail because team reported occupancy:2 while agents reported no live agents.
role: root
