# Control-plane service skeleton

This directory is the Phase 1 reference service boundary decided in
[ADR 0001](../docs/decisions/0001-control-plane-boundary.md). It contains one
API process, one worker, strict shared configuration, a checksummed PostgreSQL
migration runner, and the first tenant-scoped persistence boundary. It does
**not** yet wire browser/device-code login or tenant routes into the standalone
binary, nor ship billing, GitHub service, queue-consumer, or remote-execution
behavior. The tenant HTTP boundary is injectable so authentication and database
drivers remain explicit deployment choices rather than hidden defaults.

Neither process imports dacli's local workspace, task store, or execution
packages. The stable client/server boundary remains
[`contracts/controlplane/v1`](../contracts/controlplane/v1/README.md).

## Local topology

The development topology is deliberately small and explicit:

| Boundary | Binding | Credentials | Persistence |
| --- | --- | --- | --- |
| API | `127.0.0.1:8080` | service secret via environment | none |
| PostgreSQL 17.6 | `127.0.0.1:55432` | required environment value | named Docker volume |
| Worker | no network listener | same environment references | PostgreSQL |

Start PostgreSQL with a non-default local password:

```bash
export DACLI_CLOUD_POSTGRES_PASSWORD="$(openssl rand -hex 24)"
docker compose -f cloud/compose.yaml up -d
export DACLI_CLOUD_DATABASE_URL="postgres://dacli_control_plane:${DACLI_CLOUD_POSTGRES_PASSWORD}@127.0.0.1:55432/dacli_control_plane?sslmode=disable"
export DACLI_CLOUD_SERVICE_SECRET="$(openssl rand -hex 32)"
go run ./cloud/cmd/api --config cloud/config.development.json
```

Run the worker in another terminal with the same environment:

```bash
go run ./cloud/cmd/worker --config cloud/config.development.json
```

`sslmode=disable` is permitted only for this loopback development topology.
Production configuration requires an HTTPS public URL, a non-default service
secret of at least 32 bytes, a PostgreSQL URL that does not disable TLS, exact
contract version 1, and no unknown configuration fields.

The compose file publishes PostgreSQL only on loopback, requires the password
instead of providing a committed default, and stores database state in the
named `dacli-control-plane-postgres` volume. Removing that volume destroys the
local database and is never performed by the application.

## Migrations

Migration files use contiguous names such as `0001_service_state.sql`. The
runner hashes every byte, compares the complete applied ledger before writing,
and applies each missing migration and its ledger record in one SQL
transaction. It refuses gaps, unknown catalog files, duplicate/applied
versions, an edited checksum, and a database version newer than the binary.

A deployment must explicitly link a PostgreSQL `database/sql` driver. The
repository intentionally does not select a driver: embedders retain control of
driver version, connection pooling, TLS, and credential rotation.

Migration `0003_tenant_repository.sql` creates the tenant identity graph. Every
tenant-owned table has a composite tenant key, tenant-bearing relationships,
and enabled plus forced row-level security. Policies compare against the
transaction-local `dacli.tenant_id`; an absent setting sees and changes no
tenant rows. The audit table rejects updates and deletes with a database
trigger.

Migration `0005_tenant_workflows.sql` adds invitation and project/environment
assignment records. It stores only invitation-token digests, carries tenant
identity through every relationship, and forces row-level security on all
three tables. Tenant resources have no hard-delete repository method.

Migration `0006_stable_pagination.sql` adds immutable identity ordinals and
tenant-leading indexes for project and environment snapshot pagination. New
inserts receive larger ordinals; an in-progress traversal remains bound to its
original high-water mark.

## Tenant domain kernel

`internal/tenant` is the shared, transport-independent domain boundary. It
defines distinct opaque identifiers and versioned closed values for accounts,
organizations, teams, memberships, devices, projects, and environments. Every
tenant-owned validator requires an explicit organization scope.

Authorization reloads the current membership on every operation and requires
the caller's exact membership version. Removed, suspended, revoked, expired,
wrong-tenant, and stale-version memberships all return the same deny result.
The role matrix is closed and deny-by-default: owners receive every known
permission; administrators omit billing; managers govern teams, memberships,
projects, and environments; developers can use devices and write projects;
reviewers are read-only; billing is limited to organization/billing access;
auditors receive organization/project/audit read access. Unknown roles and
permissions never inherit access.

Audit records are pointer-free values binding tenant, actor/device, action,
target, optimistic before/after versions, fixed SHA-256 values, and occurrence
time.

## Tenant repository

`internal/tenantrepo` opens only against the exact migration version understood
by the binary. Every operation requires an explicit `tenant.Scope`, begins a
database transaction, sets the matching transaction-local RLS identity, and
also carries the tenant in each predicate. Cross-tenant and missing identifiers
therefore return the same result rather than disclosing existence.

Project writes use exact optimistic versions and append their immutable audit
event before the same transaction commits. A failed state write, audit append,
or commit leaves neither half visible. Membership reads use the same boundary
and implement `tenant.MembershipSource`, so authorization always reloads current
tenant state rather than trusting an HTTP claim or cache.

## Tenant resource workflows

`internal/tenantresource` is the provider-neutral service boundary for
invitations, projects, environments, and their account assignments. Every
privileged operation reloads the actor's current membership and asks for the
one exact permission: membership management for issue/cancel/expire, project
management for project lifecycle and assignment, and environment management
for environment lifecycle and assignment. A denied or stale membership never
reaches persistence.

Invitation acceptance is the deliberate exception to membership authorization:
the invited account does not have a membership yet, so a minimum-strength
one-time credential and its bound opaque account identity authorize that one
transition. The raw credential is hashed before it reaches the repository.
Acceptance locks and advances the pending invitation, creates the membership,
and appends both audit facts in one transaction; expiry, cancellation, replay,
and concurrent acceptance fail closed. Projects, environments, and assignments
use exact version increments, and removal is represented by recoverable
archive/restore state transitions rather than deletion.

These records use closed structs and explicit SQL columns. They retain only
opaque IDs, names, roles, lifecycle, versions, and expiry. They cannot accept
source, diffs, prompts, transcripts, command output, paths, secrets,
environment values, or arbitrary metadata, matching the v1 privacy boundary.

## Bounded API, cache, and worker scope

`internal/tenantapi` composes current-membership authorization with the
repository. It refuses a stale policy revision before any database or cache
access. `internal/service` exposes bounded project and environment routes only
when an identity verifier, backend, and cursor key are explicitly installed.
The temporary HMAC identity is short-lived and is replaced by native device
login in #984. Request bodies cannot carry tenant identity.

Project and environment lists use monotonic high-water snapshots, a maximum
page size of 100, and HMAC-bound opaque cursors. The bounded tenant cache keys
tenant, kind, resource, subject, resource/membership versions, and policy
revision; authorization is reloaded before cache access. The worker dispatcher
likewise derives scope only from its verified identity and ignores an untrusted
claimed tenant. See [the API boundary](../docs/CONTROL_PLANE_API.md).

## Device sessions

`internal/devicesession` owns provider-neutral device authorization. Raw
credentials are accepted only at method boundaries and immediately reduced to
SHA-256 digests; records deliberately omit the digest from JSON. Every
authorization reloads the tenant-scoped session, device, and membership and
then uses a constant-time digest comparison. Expiry, suspension, membership
removal, device/session revocation, wrong-tenant use, and stale rotation are
indistinguishable denials.

Migration `0004_device_sessions.sql` stores only fixed-size credential digests,
uses composite tenant relationships and forced RLS, and uniquely scopes a
digest within its tenant. Rotation revokes the old record, creates the new
record, and appends both audit facts in one transaction; optimistic versions
make a replay or second revocation a conflict before side effects.
