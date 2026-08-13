# Signed control-plane event protocol v1

This directory is the wire contract between a dacli workspace adapter and an
optional remote control plane. It exchanges allowlisted metadata only. The
local markdown workspace remains authoritative: inbound task, approval, and
policy events are proposals for the local Inbox and cannot directly mutate a
task, start an agent, grant a capability, or replace the append-only local
event log.

## Envelope and integrity

Each Outbox record is an immutable `envelope.schema.json` document whose
`payload` validates against the schema named by `event_type`. Producers assign
a strictly increasing `producer_sequence`; consumers persist the event and
idempotency indexes before acknowledging it. The signature is Ed25519 over
RFC 8785 canonical JSON of the complete envelope with `integrity.signature`
set to the empty string. Verification selects the tenant-scoped public key by
`integrity.key_id`; an unknown or revoked key is a tampered outcome.

The Inbox is append-only. Process in this order: verify tenant/project routing
and signature, negotiate schema compatibility, detect `event_id` or
`idempotency_key` duplicates, enforce the retained replay floor, then compare
the producer sequence. Gaps and reordered records may be stored and applied
when their object `base_version` matches. A mismatch is an optimistic-version
conflict requiring a new proposal; it never overwrites local state.

## Metadata allowlist

The payload schemas are closed with `additionalProperties: false`; their named
properties are the complete v1 allowlist. Source code, prompts, transcripts,
command output, environment names or values, credentials, tokens, and secrets
are therefore excluded by default. Adding any such class requires a new schema
version and an explicit disclosure policy. Implementations must also reject
unexpected nested fields rather than preserve or forward them.

## Compatibility, migration, and retention

- A v1 consumer accepts only schema version `1`. Unknown newer versions are
  retained as incompatible but are not interpreted or acknowledged as applied.
- Once a producer has emitted or a consumer has acknowledged version N, it
  refuses downgrade to a lower version. Migration emits a new immutable event;
  stored envelopes are never rewritten in place.
- Inbox envelopes and idempotency keys are retained for at least 90 days.
  Per-producer sequence cursors and the replay floor are retained indefinitely,
  including after payload expiry, so expired events cannot be replayed.
- Offline producers keep an ordered Outbox until acknowledgement. On reconnect
  they replay from the last acknowledged cursor with original ids, sequences,
  timestamps, and signatures. Delayed and reordered events remain deterministic;
  cursor advancement waits for gaps or an explicit operator-approved skip.
- Consumers reject events below the replay floor, invalid signatures, wrong
  tenant/project routing, and incompatible versions. They do not downgrade,
  reinterpret, or silently discard them.

The golden fixtures in `testdata` make duplicate, reordered, delayed, replayed,
incompatible, and tampered handling executable. Their outcomes are protocol
terms, not instructions to mutate the workspace.
