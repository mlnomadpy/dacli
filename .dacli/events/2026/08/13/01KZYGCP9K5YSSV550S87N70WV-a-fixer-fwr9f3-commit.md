---
id: 01KZYGCP9K5YSSV550S87N70WV
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-13T21:27:59Z
created_by: a-fixer-fwr9f3
about: "[[t-01KZYA0JT4Q7PM1F1PEHRFFVSF]]"
origin: agent
applied: true
checksum: sha256:76bd82d4b445b9fbdf380cbd5b322e1eb54487d5505cc44dd5a6a9b21f431aac
---
e41bfb2 433: export stable scenario metrics JSON

Centralize completion, retry, failure class, wall time, token usage and budget, and intervention samples in one report consumed by insight and wscore. Missing token data remains null instead of becoming zero.

Mutation proof: removing failure aggregation makes TestCompareNamedScenarioWindowsRejectsMissingOrFabricatedData fail with: failure class data missing: {Classes:map[] Samples:0}.
role: fixer
