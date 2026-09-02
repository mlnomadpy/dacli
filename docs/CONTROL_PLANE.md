# Hosted control-plane implementation

**Status: development skeleton shipped in source; hosted product not shipped.**

The [`cloud/`](https://github.com/mlnomadpy/dacli/tree/main/cloud) boundary now
contains a runnable API lifecycle, background-worker lifecycle, strict typed
configuration, and a transactional checksummed PostgreSQL migration runner.
This is infrastructure for the Phase 1 plan, not a customer-ready SaaS claim.
There is no tenant membership, device login, metadata synchronization, billing,
approval service, portfolio view, GitHub App service, hosted deployment, or SLO
yet.

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
- Linux CI coverage and an import-boundary test preventing the cloud service
  from coupling to local task-store or coding-agent execution internals.

The exact development commands and port, credential, volume, and migration
boundaries are in the
[`cloud` README](https://github.com/mlnomadpy/dacli/blob/main/cloud/README.md).

## What comes next

The program remains tracked by
[#446](https://github.com/mlnomadpy/dacli/issues/446). The ordered implementation
path is tenant isolation, signed device authentication, metadata-only sync,
signed role/policy publication, approval and budget controls, the GitHub App
service, and finally the cross-project operator view. Each feature must satisfy
the [threat model](CONTROL_PLANE_THREAT_MODEL.md),
[privacy boundary](CONTROL_PLANE_PRIVACY.md), and public
[`controlplane/v1`](../contracts/controlplane/v1/README.md) contract before it
can be described as shipped.
