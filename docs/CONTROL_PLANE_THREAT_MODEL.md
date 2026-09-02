# Control-plane threat model

**Status: Phase 1 design boundary; the hosted service is not shipped.** This
model governs issues #982, #984, #981, #979, #978, #980, and #983. It does not
claim a security audit, compliance certification, or production readiness.

## Assets and actors

Highest-impact assets are tenant signing keys, device refresh credentials,
GitHub App private keys and installation tokens, approval/action hashes,
released role and policy bundles, tenant membership, budget authority, audit
history, and the mapping between local projects and remote tenants. The system
must also protect customers from unintended disclosure of source, prompts,
transcripts, environment data, command output, and secrets—which are not valid
v1 payload fields at all.

Actors include tenant owners/admins, managers, developers, reviewers, billing
admins, auditors, service accounts, local coding agents, the hosted operator,
GitHub, a network attacker, a malicious tenant member, a compromised local
device, a compromised agent runtime, and a malicious external issue/comment
author. Authentication establishes identity; it does not make remote prose a
trusted instruction.

## Trust boundaries and entry points

```text
untrusted repository/GitHub prose
              |
              v
local dacli -- signed, closed metadata --> HTTPS API --> tenant services --> PostgreSQL
     |                 device auth             |              |
coding CLI                              inbox/outbox worker    audit stream
                                                            |
GitHub webhook --> signature-first bridge --> proposals/check summaries
```

Entry points are device authorization, metadata sync, web/API sessions,
GitHub webhooks, installation-token API calls, background jobs, exports, and
operator administration. Browser, API, worker, database, cache, object store,
GitHub, and every local device are separate trust zones. Tenant and project
scope must be rechecked in repository/service methods and job payloads, not
accepted from HTTP middleware or a cache key.

## Required controls

| Threat | Required mitigation and failure behavior |
|---|---|
| Cross-tenant object reference or cache collision | Tenant-scoped repository APIs, composite keys/RLS where appropriate, negative cross-tenant fixtures at API/service/repository/job/cache/object-store layers; deny before existence disclosure. |
| Forged, replayed, reordered, or downgraded event | Verify tenant/project route and Ed25519 signature before schema/idempotency/replay checks; retain replay floors; refuse incompatible versions; never rewrite an accepted envelope. |
| Credential leakage | Native OS credential store locally; managed secret/key storage remotely; short-lived installation tokens; never place credentials in arguments, payloads, logs, errors, fixtures, or `.dacli`. |
| Malicious issue, comment, prompt, or agent output | Treat all prose as attributed data; translate remote requests into proposals; deterministic policy owns authority; never feed webhook text as system instructions. |
| Approval substitution or replay | Bind decision to tenant, project, requester, exact action hash, scope, version, approver, and expiry; one-time consumption is atomic and auditable. |
| Excess collection | Closed schemas plus `privacy-fields.json`; reject unknown/nested fields client-side before signing and server-side before persistence; size/count/request limits. |
| GitHub permission expansion | Selected-repository installations, least-privilege manifest, permission recheck before dispatch, suspension/removal reconciliation; remote discovery cannot expand local tenant binding. |
| Queue duplication or crash window | Durable inbox and idempotency index before acknowledgement; transactional outbox; stable delivery IDs; bounded retry and queryable dead letter. |
| Resource exhaustion or analytics cardinality abuse | Auth/login/sync/webhook/list rate limits, payload and page bounds, timeouts, cancellation, aggregation limits, and backpressure. |
| Audit deletion or equivocation | Append-only tenant events with integrity chaining/export verification, retention policy, restricted operator access, and compensating records rather than edits. |
| Dependency or build compromise | Locked dependencies, SBOM/provenance, secret/SAST/dependency/container scans, protected release workflow, and reproducible migration/contract tests. |

## Security invariants

- Local markdown remains authoritative for local execution; inbound cloud and
  GitHub data cannot directly start agents, mutate tasks, or grant capability.
- No signed bundle is equivalent to authority unless it is current, compatible,
  correctly scoped, unrevoked, and accepted by local policy.
- Required cloud policy fails closed when configured and missing, expired,
  invalid, or incompatible. No cloud configuration means local-only behavior,
  not a hidden network dependency.
- Metadata collection is deny-by-default. A new field requires a schema version,
  privacy classification, client/server validation, migration, and disclosure.
- Production side effects require exact tenant/project identity and a fresh
  authorization check at the repository/service boundary.

## Residual risks and readiness gate

A compromised local device can observe data available to that device and can
attempt validly signed false metadata until revoked. A compromised hosted
operator or database layer remains high impact despite encryption. Metadata can
still reveal repository names, work timing, role/model choices, and team
structure. Aggregates can deanonymize small samples. GitHub and identity
providers remain external dependencies.

Before accepting customer repositories, the implementation must complete
cross-tenant tests, key/device revocation, backup/restore, deletion/retention,
incident response, external security review of auth/crypto/tenant boundaries,
and explicit design-partner consent for the enabled privacy fields. Passing
this document alone is not that gate.
