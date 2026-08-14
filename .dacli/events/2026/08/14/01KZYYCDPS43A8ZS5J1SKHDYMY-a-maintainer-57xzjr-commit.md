---
id: 01KZYYCDPS43A8ZS5J1SKHDYMY
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-14T01:32:30Z
created_by: a-maintainer-57xzjr
about: "[[t-01KZYXX2BZPRTWR5Z982B6SM76]]"
origin: agent
applied: true
checksum: sha256:a97417bee3cc390d68dacf377c8036f8186b1b33267f6eb468a3041173ed0aac
---
2769d8b 453: select PR path for configured record tail

The loop already resolved the durable project landing policy, but its record-only ship call omitted --pr whenever that policy was configured rather than explicitly overridden. Select ship's PR-capable path while retaining no-accept/no-integrate and leaving the configured base for ship to resolve.

Mutation proof: TestRecordSelfPRSelectsConfiguredPRPathWithoutInventingOverride failed with "configured PR record call omitted --pr: [ship --no-accept --no-integrate --project p --push]".
role: maintainer
