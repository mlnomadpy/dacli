# Control-plane service skeleton

This directory is the Phase 1 reference service boundary decided in
[ADR 0001](../docs/decisions/0001-control-plane-boundary.md). It contains one
API process, one worker, strict shared configuration, and a checksummed
PostgreSQL migration runner. It does **not** yet ship tenant, authentication,
billing, GitHub integration, queue-consumer, or remote-execution behavior.

Neither process imports dacli's local workspace, task store, or execution
packages. The stable client/server boundary remains
[`contracts/controlplane/v1`](../contracts/controlplane/v1/README.md).

## Local topology

The development topology is deliberately small and explicit:

| Boundary | Binding | Credentials | Persistence |
| --- | --- | --- | --- |
| API | `127.0.0.1:8080` | service secret via environment | none |
| PostgreSQL 17.6 | `127.0.0.1:55432` | required environment value | named Docker volume |
| Worker | no network listener | same environment references | PostgreSQL (future adapter) |

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
standard-library skeleton intentionally does not select or silently download a
driver; the API and worker do not claim database-backed readiness until that
adapter is added with the domain work.

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
time. Later persistence work must append that value in the same transaction as
the state change; this package itself performs no I/O.
