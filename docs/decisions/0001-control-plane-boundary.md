# ADR 0001: Keep the first control-plane service in this repository

- Status: accepted for Phase 1
- Date: 2026-09-02
- Program: [#446](https://github.com/mlnomadpy/dacli/issues/446)
- Decision gate: [#989](https://github.com/mlnomadpy/dacli/issues/989)

## Context

dacli already ships a local, offline-capable agent control plane, a signed
metadata protocol, an offline sync client, and a constrained GitHub App domain
bridge. The subscription opportunity is hosted coordination across projects,
not a proprietary coding model and not secrecy around the protocol. Splitting
the first server before its domain is validated would make every contract
change cross-repository, weaken executable conformance, and create release and
security overhead without an independent scaling boundary.

## Decision

Build the Phase 1 reference control-plane service as a modular monolith under
`cloud/` in this MIT-licensed repository. Use one API process, one worker,
PostgreSQL, and optional object storage only for artifacts a tenant explicitly
enables. Keep `contracts/controlplane/v1/`, the local sync client, and their
golden compatibility tests public permanently.

The hosted subscription sells operation: managed availability, tenant
administration, shared governance, approvals, portfolio visibility, retention,
audit exports, and support. It does not depend on concealing the server source.
Core local commands remain useful without an account.

## Primary MVP persona and paid workflow

The primary MVP persona is an AI-platform or engineering lead already using
multiple coding-agent CLIs across at least two repositories. The first complete
paid workflow is:

1. register an organization, device, and two projects;
2. publish an exact signed role and policy version;
3. synchronize allowlisted run/gate/agent/budget metadata;
4. see stale, blocked, over-budget, or approval-waiting work across projects;
5. approve one exact hashed action and let only that action resume locally.

Remote execution, bundled model usage, IDE/source browsing, transcript upload,
SAML/SCIM, and Kubernetes are not part of this workflow.

## Rejected alternatives

- **Private service repository now.** Rejected until a licensing, contractual,
  or security boundary outweighs atomic contract tests and public review.
- **Separate microservices.** Rejected until a module demonstrates a measured
  independent scaling, security, data-sovereignty, or failure boundary.
- **GitHub as the source of truth.** Rejected because local execution must stay
  offline, fast, append-safe, and independent of API quotas.
- **Remote runners in the MVP.** Rejected because they multiply credential,
  sandbox, artifact, network, billing, and incident-response risk before the
  coordination product is validated.

## Migration triggers

Moving deployable `cloud/` code to a separate repository requires a recorded
decision and at least one concrete trigger: incompatible licensing, regulated
access requiring a narrower contributor set, independent release cadence that
repeatedly blocks the local CLI, or measured scaling/failure isolation that the
modular monolith cannot meet. A split must keep public schemas, privacy
manifest, compatibility policy, and golden fixtures here and run them against
both repositories in CI.

Splitting an individual service additionally requires measured load or a
security/data boundary, a defined ownership and SLO, a versioned protocol, and
a rollback plan. Team size or architectural fashion is not a trigger.

## Consequences

Contract and server changes can land atomically and undergo one security review.
The repository becomes larger and cloud modules must be prevented from
importing local workspace/execution internals. Hosted secrets, customer data,
deployments, and production configuration never belong in git. The bounded
API/worker/configuration/migration skeleton now exists under `cloud/`; the
multi-tenant hosted service remains unshipped until the separate domain,
security, and deployment gates are complete. This ADR is authority to build,
not a customer-readiness claim.
