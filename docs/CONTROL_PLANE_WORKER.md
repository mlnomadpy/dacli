# Durable control-plane envelope worker

**Status: implemented provider-neutral persistence and delivery boundary; not wired to a hosted deployment.**

`cloud/internal/envelopeworker` is the server-side counterpart to the local
`internal/cloudsync` client. It accepts and emits only signed
[`controlplane/v1`](../contracts/controlplane/v1/README.md) envelopes. It does
not interpret an inbound proposal, mutate a local task, or start an agent.

## Inbound order and acknowledgement

The outer HTTP/queue adapter must enforce its body limit before constructing a
request. The service then performs these checks in order:

1. authenticate the opaque credential into a closed `VerifiedIdentity`;
2. compare the envelope route with that verified tenant and validate bounded
   envelope identities;
3. resolve keys inside the verified scope and verify the Ed25519 signature;
4. require schema version 1 and validate the closed metadata payload;
5. lock the tenant/project/producer stream and check retained event and
   idempotency identities, replay floor, and producer sequence;
6. commit the permanent identity, expiring payload, cursor advancement, and
   redacted audit event in one transaction;
7. only then return an acknowledgement outcome.

Authentication failures cannot enter a tenant audit stream because they have
no verified tenant or actor. Every authenticated accept/refusal records tenant,
project, actor/device, a SHA-256 event identity, outcome, stable reason, time,
and correlation ID. It never records the credential, signature, or payload.

Deployments construct the inbound service with the configured `sync` limiter.
The limiter runs after authentication and before route/key/signature/database
work, keyed only by verified tenant/account/device plus the `sync-ingest`
purpose. Authentication adapters must apply the separate peer-keyed identity
limit before expensive credential verification.

Duplicates are acknowledged only after their duplicate audit commits. Gaps do
not advance the contiguous cursor; a later missing sequence advances through
all now-contiguous retained identities. Reordered records are retained under
the public v1 outcome. A failed identity, payload, cursor, audit, or commit
returns no acknowledgement, so retry is safe.

## Outbox delivery

Signed envelopes enter the outbox under a stable idempotency key. Reusing that
key returns the original immutable envelope; colliding content is refused.
Workers claim at most 100 due rows with `FOR UPDATE SKIP LOCKED` and a bounded
lease. An expired lease makes a crash recoverable. Delivery uses caller-set
maximum attempts, exponential capped backoff, bounded jitter, and context
cancellation. Success, retry, and dead-letter transitions compare the claimed
attempt counter so a stale worker cannot acknowledge a newer lease.

Dead letters are queryable by explicit tenant/project scope with a maximum page
of 100. Diagnostics retain only a stable error code, never a provider response,
token, or envelope payload.

Delivery uses the same independently configured `sync` policy but a distinct
trusted tenant-scheduler `delivery` key. It checks cancellation and rate before
claiming a lease, so exhaustion cannot strand rows in `delivering`. Rate
limiting does not change the 100-row claim bound, retry cap, lease recovery, or
dead-letter semantics.

## Retention and recovery

Inbox payloads expire after 90 days by default and are deleted in batches of at
most 1,000. The cleanup query can delete only the payload table. Event IDs,
idempotency keys, producer sequences, contiguous cursors, replay floors, and
audit events remain, so payload expiry cannot make an old event new again.

Database backup/restore must preserve all envelope tables at one PostgreSQL
snapshot. Restoring payloads without identities/cursors is invalid. Signing-key
rotation adds a new tenant-scoped key ID; old public keys remain verifiable for
the retention/replay window and revoked keys are removed only after their
accepted envelopes can no longer be replayed. Recovery should resume expired
leases and inspect dead letters; it must not reset attempts, cursors, replay
floors, or idempotency identities.

The checked-in worker command owns cancellation and cycle timeouts, but a
deployment must explicitly inject its PostgreSQL driver, tenant scheduler,
credential verifier, tenant-scoped signing-key source, and network sender.
Until that composition exists, the repository does not claim a running hosted
queue.
