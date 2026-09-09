# Bounded tenant API boundary

**Status: implemented as an injectable Phase 1 boundary; hosted deployment and
native device login are not shipped.**

The control-plane HTTP service can enable tenant routes only when it is given
three explicit dependencies: an identity verifier, a tenant backend, and at
least 32 bytes of cursor-signing key material. Health and readiness do not
pretend these routes are active when those dependencies have not been wired.

## Identity and scope

The temporary Phase 1 verifier accepts a short-lived, HMAC-authenticated bearer
identity containing an opaque tenant, account, optional device, exact
membership version, current policy revision, and expiry. Domain separation
prevents an identity signature from being reused as a page-cursor signature.
Native device-session authentication replaces this adapter in #984.

Handlers use only the verified identity to construct `tenant.Scope`, actor, and
device. A body, query, route, or queued work item may select a resource inside
that scope but cannot supply or replace the tenant. The application layer
reloads membership authorization before repository or cache access and rejects
a stale policy revision before either one. Unknown, removed, stale, and
cross-tenant resources share one `resource_unavailable` response.

## Routes

| Method and path | Contract |
| --- | --- |
| `GET /v1/projects?limit=&cursor=` | Stable snapshot page, default 25 and maximum 100 |
| `POST /v1/projects` | Closed ID/name body; active version-one creation |
| `GET /v1/projects/{id}?version=` | Exact-version read; authorization is reloaded even on a cache hit |
| `PATCH /v1/projects/{id}` | Closed name/state/expected-version body; optimistic audit-atomic update |
| `GET /v1/projects/{id}/environments?limit=&cursor=` | Stable project-bound environment page |
| `POST /v1/projects/{id}/environments` | Closed ID/name/kind body; active version-one creation |
| `PATCH /v1/projects/{id}/environments/{environment}` | Closed lifecycle update with exact expected version |

PATCH bodies reject unknown fields, multiple JSON values, invalid versions, and
chunked bodies that exceed the global request limit. Structured errors never
include database, credential, or object-existence detail.

## Mutation attempt evidence

After identity verification, every project/environment mutation either commits
its success audit atomically with state or appends a separate non-success audit
after the failed state transaction rolls back. Closed outcomes distinguish
authorization and invalid-state refusals, unavailable resources, version
conflicts, and persistence failures without changing the public
missing/cross-tenant response. If that audit append itself fails, the handler
returns a retryable dependency error and does not claim durable refusal
evidence. The request ID is the correlation identity: an exact replay is
idempotent, while reuse for a different actor, device, action, result, or reason
fails closed.

For recovery, inspect the tenant audit row by request/correlation ID before a
retry. Re-submit only the identical authenticated action when its recorded
outcome and current resource version make that safe; generate a new request ID
for a changed action. An audit-correlation conflict is evidence of ambiguous or
reused identity and requires operator investigation, never an overwrite or an
automatic retry with altered input.

## Stable pagination

Migration `0006_stable_pagination.sql` assigns immutable, monotonically
increasing ordinals to projects and environments. The first query captures a
tenant-scoped high-water ordinal in the same read transaction. Later pages use
`after < ordinal <= snapshot`, ascending order, and one bounded lookahead row.
Inserts after the snapshot therefore appear only in a new traversal and cannot
create a duplicate or gap in the current traversal.

The cursor is an opaque base64url payload plus HMAC. Its signature binds tenant,
resource kind, optional parent project, snapshot, and last returned ordinal.
Cross-tenant, cross-kind, cross-parent, malformed, and modified cursors are
rejected before the repository is called.

## Cache and worker boundary

The bounded LRU cache key contains tenant, resource kind and ID, authenticated
subject, resource version, membership version, and policy revision. Every
dimension participates in equality. Authorization is always reloaded before a
cache lookup, so revocation takes effect on the next operation; mutations
invalidate that tenant's cached projects.

The worker dispatcher follows the same rule. Its verifier produces the
identity and scope, while an untrusted claimed tenant remains data only. Invalid
identity never reaches the processor, and the raw worker credential is excluded
from JSON.
