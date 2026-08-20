---
id: 01M0F2GMRC14RMCK7N8JNENGV5
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-20T07:52:34Z
created_by: a-root
about: "[[t-01M0CX03Q4A1BM8JD9YQBCNGV0]]"
origin: agent
applied: true
checksum: sha256:53a63669ad2adf0f8f6f6f5e57f51fcfb3cb0be3aeeb13eb21ffc749498655f9
---
057c7c4 t-01M0CX03Q4A1BM8JD9YQBCNGV0: harden profile grants, refusals, and operator guidance

Keep inspect/show readable without allowing mutating profiles through the grant gate, stop service supervision immediately on exit 3, and teach the dacli skill the five shipped operating profiles.

Mutation: returning false for a clikit refusal made TestServiceNeverRetriesPolicyRefusal fail with calls=3 instead of calls=1.
role: root
