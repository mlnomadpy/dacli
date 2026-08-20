---
id: 01M0D9Z7T4JBZR6PNFGH7Y29K7
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-19T15:24:23Z
created_by: a-maintainer-3necr2
about: "[[t-01M0CX03Q4A1BM8JD9YQBCNGV0]]"
origin: agent
applied: true
checksum: sha256:162b2dac27676c91859a8373f61b1f4f327646426339d3f9cf510c2fc5dbb9db
---
125245b t-01M0CX03Q4A1BM8JD9YQBCNGV0: add bounded operating profiles and service supervision

Resolve and persist provider-neutral inspect/task/wave/loop/service policy, delegate execution to existing loop strategies, and stop service runs at durable lease, STOP, budget, landing, and circuit-breaker checkpoints. Release authority remains separate and off.

Mutation: setting default RollingTokens to 0 makes TestOperatingProfileGoldenDefaultsAreFiniteAndReleaseIsOff fail at profile_test.go:35 with 'task has an unbounded default'.
role: maintainer
