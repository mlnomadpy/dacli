# Hosted control-plane implementation

**Status: development skeleton shipped in source; hosted product not shipped.**

The [`cloud/`](https://github.com/mlnomadpy/dacli/tree/main/cloud) boundary now
contains a runnable API lifecycle, background-worker lifecycle, strict typed
configuration, a transactional checksummed PostgreSQL migration runner, and a
tenant-scoped repository foundation. This is infrastructure for the Phase 1
plan, not a customer-ready SaaS claim. There is no browser/device-code login
CLI, deployed tenant API, metadata synchronization, billing, approval service,
portfolio view, hosted GitHub App service, deployment, or SLO yet.

## What exists

- one bounded HTTP process with health/readiness endpoints, request IDs,
  structured safe errors, request limits, timeouts, and graceful shutdown;
- one cancellable worker process with immediate and periodic bounded cycles;
- configuration that resolves credentials from named environment variables,
  rejects unknown fields, and fails closed on unsafe production transport,
  default credentials, or incompatible contract versions;
- contiguous SQL migrations with SHA-256 ledger comparison and one transaction
  per migration plus ledger write;
- a loopback-only PostgreSQL 17.6 development topology with required credentials
  and an explicit named volume;
- a transport-independent tenant kernel with scoped versioned entities,
  deny-by-default role authorization, current-membership revalidation, and
  immutable digest-bound audit values;
- a PostgreSQL tenant schema with composite tenant keys, tenant-bearing foreign
  keys, forced row-level security, and database-enforced append-only audit
  events;
- canonical successful-mutation action identities binding scope, operation,
  target, optimistic versions, and before/after state, with explicit
  `succeeded`/`committed` evidence in the append-only audit stream;
- an exact-schema repository boundary that binds the tenant into both the
  transaction-local RLS setting and every query, uses optimistic versions, and
  commits project mutations with their audit event atomically;
- revocable device-session authorization that stores only one-way credential
  digests, reloads current session/device/membership state for every protected
  operation, and rotates or revokes with optimistic transactional audit;
- a provider-neutral tenant-resource workflow for digest-only one-time
  invitations, project/environment lifecycle, and scoped account assignments;
  exact permissions are checked against current membership before persistence,
  while every mutation is optimistic and audit-atomic;
- injectable HTTP and worker tenant boundaries that derive scope only from a
  short-lived verified identity, reject stale policy revisions, use signed
  tenant/parent-bound snapshot cursors, and isolate a bounded versioned cache;
- a durable signed-envelope inbox/outbox boundary that commits permanent
  idempotency identity, expiring payload, cursor, and redacted audit before
  acknowledgement, plus leased bounded retry and queryable dead letters;
- Linux CI coverage with explicit floors for both process entrypoints,
  configuration, migrations, service, tenant, and worker packages, plus an
  import-boundary test preventing the cloud service from coupling to local
  task-store or coding-agent execution internals.

The exact development commands and port, credential, volume, and migration
boundaries are in the
[`cloud` README](https://github.com/mlnomadpy/dacli/blob/main/cloud/README.md).

## What comes next

The program remains tracked by
[#446](https://github.com/mlnomadpy/dacli/issues/446). The ordered implementation
path is native signed device authentication, metadata-only sync, signed
role/policy publication, approval and budget
controls, the GitHub App service, and finally the cross-project operator view.
Each feature must satisfy
the [threat model](CONTROL_PLANE_THREAT_MODEL.md),
[privacy boundary](CONTROL_PLANE_PRIVACY.md), and public
[`controlplane/v1`](../contracts/controlplane/v1/README.md) contract before it
can be described as shipped.

The exact persistence, crash, retry, retention, and recovery contract is in the
[worker boundary](CONTROL_PLANE_WORKER.md).
