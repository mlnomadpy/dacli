---
id: 01M0N02NKAT61FXWERHZ0PZR76
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-22T15:05:25Z
created_by: a-root
about: "[[t-01M0MZ05DNTWBYQSTYP5X8J2NF]]"
origin: agent
applied: true
checksum: sha256:3eef591f557b88e2a8e39de89019488f87bca59e9ec37983ccc4506b09c0fe2a
---
b16a5e2 task 493: stop Codex preflight at readiness

Version the Codex JSONL probe, recognize turn.started without waiting for inference, and reap the bounded process tree. Document the exact cache and cost semantics.

Mutation: changing the readiness predicate to turn.ready made TestCodexBehavioralPreflightReadinessStopsAndReapsHangingTree fail with transient/transport deadline exceeded.
role: root
