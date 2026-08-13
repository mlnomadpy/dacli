---
id: t-01KZX7PXXH56GVGAWEG2H7JDPV
kind: task
created: 2026-08-13T09:37:03Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
github:
  issue: 552
  repo: mlnomadpy/dacli
---
# Define the signed control-plane event protocol
## So that
the optional SaaS and GitHub App exchange metadata without making the cloud the workspace source of truth
## Context
Use an append-only event envelope, Inbox/Outbox, idempotency keys, optimistic versioning, and anti-corruption boundaries. The local markdown workspace remains authoritative for agent execution.
## Acceptance
- [ ] contracts/controlplane/v1 defines versioned schemas for installation, repository, task proposal, run summary, approval, policy bundle, and sync cursor
- [ ] An envelope carries schema version, tenant and project ids, event id, producer sequence, timestamp, idempotency key, producer version, and integrity signature
- [ ] A field allowlist excludes source code, prompts, transcripts, command output, environment values, and secrets by default
- [ ] Golden fixtures assert distinct deterministic outcomes for duplicate, reordered, delayed, replayed, incompatible, and tampered events
- [ ] Compatibility and migration policy enumerates accepted schema versions, downgrade refusal, retention, and offline replay behavior
## Log
