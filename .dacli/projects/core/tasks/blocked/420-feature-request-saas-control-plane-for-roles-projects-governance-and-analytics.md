---
id: t-01KZXS3QKA4MHMQZNK7B9VQ7V5
kind: task
created: 2026-08-13T14:41:08Z
created_by: a-root
owner: a-root
github:
  issue: 446
  repo: mlnomadpy/dacli
---
# Feature request: SaaS control plane for roles, projects, governance, and analytics
## Context
Adopted from GitHub issue #446.

## Feature request

Build a subscription SaaS control plane around dacli for teams operating autonomous coding agents across multiple repositories.

The product should not become another proprietary coding agent. dacli should remain the model- and runtime-neutral governance layer that coordinates Codex, Claude Code, Copilot, Gemini, local models, and future agent runtimes.

> **Product promise:** Define how agents work, govern execution, and understand what is happening across every repository.

## Why this is worth building

The local CLI already has the hard and differentiated foundation: role-driven execution, agent coordination, event logs, acceptance gates, budgets, Git/worktree lifecycle, MCP, and self-hosting. Teams need a persistent layer above individual repositories for:

- Shared roles and workflows
- Cross-project visibility
- Organization policies
- Budget governance
- Human approvals
- Audit history
- Performance analytics
- GitHub integration
- Reusable engineering knowledge

Saving prompts alone is not enough to support a subscription. The recurring value is operating and governing a fleet of agents across an engineering organization.

## Product boundary

### Keep local and available in the open-source runtime

- Repository and source-code access
- Git operations and worktrees
- Command execution
- Local prompts and context
- Environment variables and secrets
- Model/runtime invocation
- Tests and acceptance gates
- Local append-only event log
- Offline execution
- Local dashboards and reports

The local product must remain useful without a SaaS account.

### Put in the SaaS control plane

- Organizations, teams, users, and service accounts
- Project/repository registry
- Versioned role catalog
- Shared workflows, skills, and policy templates
- Cross-project run/event summaries
- Approval and escalation inbox
- Central budgets and usage policies
- GitHub App integration
- Audit logs and compliance exports
- Agent/model/role performance analytics
- Subscription and entitlement management
- Optional managed remote runners in a later phase

### Do not include in the initial product

- A proprietary foundation model
- A general-purpose LLM gateway
- Mandatory source-code upload or indexing
- Managed cloud development environments
- Kubernetes or multiple microservices
- Remote agent execution before the control plane has active customers
- Unlimited bundled model usage

## Proposed architecture

Start as a modular monolith. Split services only after a module has a demonstrated independent scaling, security, or failure boundary.

```text
dacli local runtime
    │
    │ signed, versioned event/command protocol
    ▼
Control-plane API
    ├── Identity and organizations
    ├── Projects and environments
    ├── Role/workflow registry
    ├── Policies and budgets
    ├── Runs and event summaries
    ├── Approvals and escalations
    ├── GitHub integration
    ├── Analytics
    └── Billing/entitlements
         │
         ├── PostgreSQL
         ├── Object storage for explicit artifacts
         └── Background worker for webhooks/aggregation
```

### Proposed repository layout

```text
contracts/controlplane/v1/       # Canonical API/event schemas and fixtures
internal/cloudsync/              # Optional local sync client
internal/cloudidentity/          # Device login and credential storage
internal/remotepolicy/           # Signed policy bundles and local evaluation
internal/approvals/              # Approval request/result domain model
internal/telemetry/              # Privacy-filtered metrics/events
cmd/dacli/                       # login, sync, org, project, roles commands

cloud/
  cmd/server/                    # API process
  cmd/worker/                    # Webhooks and aggregate jobs
  internal/auth/                 # Users, sessions, OAuth, service accounts
  internal/organizations/        # Organizations, teams, membership, roles
  internal/projects/             # Repositories and environments
  internal/roles/                # Role versions and releases
  internal/policies/             # Inheritance and evaluation
  internal/runs/                 # Run/event summaries
  internal/approvals/            # Approval state machine
  internal/githubapp/            # Installation, webhooks, PR/check status
  internal/analytics/            # Aggregations and reports
  internal/billing/              # Plans, limits, Stripe state
  internal/audit/                # Immutable tenant audit events
  migrations/                    # PostgreSQL migrations
  web/                           # SaaS dashboard
```

If the SaaS code is placed in a separate private repository, `contracts/controlplane/v1/` and the local client must remain in dacli with a documented compatibility policy.

## Canonical contracts

- [ ] Define a versioned HTTPS/WebSocket protocol in `contracts/controlplane/v1/` using JSON Schema or protobuf plus human-readable examples.
- [ ] Create schemas for device registration, project registration, role bundle, policy bundle, run summary, event summary, approval request/result, budget state, agent state, gate evidence, and sync cursor.
- [ ] Add golden fixtures used by both `internal/cloudsync/` and `cloud/` contract tests.
- [ ] Include `schema_version`, tenant/project IDs, event ID, timestamp, local sequence, idempotency key, producer version, and integrity signature on every envelope.
- [ ] Specify compatibility, deprecation, migration, retry, ordering, duplication, clock-skew, and offline replay behavior.
- [ ] Never include source code, prompt contents, environment values, command stdout/stderr, or secrets by default.

## Authentication and tenant model

- [ ] Add `dacli login`, `dacli logout`, and `dacli whoami` through a browser/device authorization flow.
- [ ] Store local credentials through the OS credential store rather than `.dacli/` plaintext.
- [ ] Model organizations, teams, memberships, service accounts, devices, projects, and environments explicitly.
- [ ] Support owner, administrator, manager, developer, reviewer, billing, and read-only/auditor permissions.
- [ ] Enforce tenant and project authorization in repositories/services, not only HTTP middleware.
- [ ] Add organization invitations, membership removal, session/device revocation, API-key rotation, and audit events.
- [ ] Add rate limits and abuse protection for login, sync, webhooks, and high-cardinality analytics.
- [ ] Plan SAML/OIDC and SCIM interfaces without implementing them in the first MVP.

## Versioned role registry

A paid role must be more than a stored system prompt.

- [ ] Define a canonical role specification containing name, purpose, instructions, allowed capabilities/tools, model/runtime preferences, file/network policy, budget, required gates, escalation rules, skills/MCP servers, repository scope, and test scenarios.
- [ ] Implement immutable semantic versions such as `security-reviewer@1.7.0`.
- [ ] Support draft, released, deprecated, and revoked role-version states.
- [ ] Allow projects to pin exact versions, track available upgrades, preview diffs, and roll back.
- [ ] Add organization and team catalogs plus controlled inheritance/override behavior.
- [ ] Add import/export between cloud role versions and local role files under `.dacli/roles/`.
- [ ] Sign downloaded role bundles and verify signatures locally before use.
- [ ] Preserve the exact role version and policy bundle used by every run.
- [ ] Add a role test harness: scenario, fixture repository, permitted actions, forbidden actions, expected gates, budget, and outcome.
- [ ] Track success, cost, intervention, rejection, and regression metrics per role version.

## Project portfolio management

- [ ] Add `dacli project connect`, `disconnect`, `status`, and `sync` commands.
- [ ] Model a project separately from a repository so monorepos and multiple deployment environments are possible.
- [ ] Store repository provider, organization, default branch, local project ID, environments, assigned teams, active role versions, policy bundle, and sync state.
- [ ] Build a portfolio dashboard showing active/blocked/waiting/completed tasks across repositories.
- [ ] Show current agents, branch/worktree, task owner, runtime/model, elapsed time, budget consumption, and last heartbeat.
- [ ] Show PR/check/deployment status from GitHub without ingesting repository contents.
- [ ] Add filters by organization, team, repository, environment, status, role, runtime, model, and time range.
- [ ] Add stale-work, repeated-failure, budget-risk, and missing-review indicators.
- [ ] Add project health summaries for test gates, agent reliability, open work, and release readiness.

## Organization policies and budgets

- [ ] Define policy inheritance: organization → team → project → environment → task exception.
- [ ] Add policy rules for branch protection, direct pushes, required reviewers, required gates, maximum budget, allowed models/runtimes, network domains, tool capabilities, deployment, destructive actions, and sensitive paths.
- [ ] Make local policy evaluation fail closed when a required signed cloud policy is missing, expired, invalid, or incompatible.
- [ ] Show users the effective policy plus provenance for every inherited value.
- [ ] Require explicit justification, approver, scope, and expiry for exceptions.
- [ ] Add monthly organization/team/project budgets and per-task soft/hard limits.
- [ ] Support warnings, pauses, approval requests, and hard stops at configurable thresholds.
- [ ] Keep compute/model provider billing separate; dacli records attributed usage without pretending all providers share one unit.
- [ ] Add budget forecasts based on role/task history with confidence ranges and clear uncertainty.

## Approval and escalation inbox

- [ ] Define an approval state machine: requested, pending, approved, rejected, expired, cancelled, consumed, and superseded.
- [ ] Support approvals for budget increases, network access, tool capability, dependency installation, destructive actions, deployment, failed-gate override, PR merge, and policy exception.
- [ ] Bind every approval to organization, project, run, exact action hash, requester, policy reason, expiry, and one-time/reusable scope.
- [ ] Deliver requests through the web dashboard first; add email/Slack integrations later.
- [ ] Allow local execution to poll or receive signed approval results and verify action hashes before continuing.
- [ ] Record the request, decision, approver, evidence, comments, expiry, and resulting action in the audit log.
- [ ] Add delegated approval and separation-of-duty rules for security/production changes.

## GitHub App integration

- [ ] Create a GitHub App with least-privilege installation permissions.
- [ ] Ingest repository, branch-protection, issue, pull-request, review, check-run, and deployment metadata required for orchestration status.
- [ ] Verify webhook signatures, installation/tenant mapping, delivery IDs, retries, duplicates, reordering, and redelivery.
- [ ] Link dacli tasks/runs to issues, branches, commits, PRs, reviews, and check runs.
- [ ] Post concise run status/check summaries without exposing prompts, secrets, or raw logs.
- [ ] Allow issue labels or commands to request a dacli workflow only after policy evaluation.
- [ ] Handle installation suspension, repository removal, permission changes, renamed repositories, and deleted branches cleanly.
- [ ] Add a reconciliation job so missed webhooks cannot permanently corrupt project state.

## Analytics and operational intelligence

- [ ] Define metrics for task completion, time to completion, first-pass success, retries, rollbacks, human interventions, approval latency, gate failures, PR rejection, escaped defects, and budget accuracy.
- [ ] Attribute metrics to organization, team, project, task type, role version, runtime/model, and time period while enforcing minimum aggregation/privacy rules.
- [ ] Add per-role and per-runtime comparison views with sample counts and confidence intervals.
- [ ] Show cost per completed task, not merely raw token usage.
- [ ] Add failure taxonomy and trends: environment, dependency, test, permissions, merge conflict, model behavior, budget, user cancellation, and infrastructure.
- [ ] Add data retention controls and allow organizations to disable selected telemetry fields.
- [ ] Export machine-readable audit and analytics data for enterprise customers.
- [ ] Do not rank individual developers by agent output or create surveillance metrics.

## Dashboard surfaces

- [ ] Organization overview: projects, active work, blockers, approvals, budget, and risk.
- [ ] Project detail: current tasks, agents, branches/PRs, gates, policy, roles, history, and health.
- [ ] Run detail: timeline, state transitions, budget, approvals, evidence, retry/failure reason, and linked GitHub objects.
- [ ] Role catalog: versions, diffs, test results, adoption, performance, deprecation, and rollback.
- [ ] Policy editor: inherited/effective values, validation, dry-run, staged release, and audit history.
- [ ] Approval inbox: priority, reason, evidence, action diff/hash, scope, expiry, approve/reject.
- [ ] Analytics: outcomes, latency, interventions, cost, reliability, role/runtime comparisons, and trends.
- [ ] Billing/admin: members, teams, devices, service accounts, usage, plan limits, invoices, and retention.

Reuse domain-neutral components from `internal/features/dashboard/ui/` where appropriate, but keep local dashboard state separate from SaaS API state.

## Security and privacy requirements

- [ ] Publish a threat model before accepting customer repositories.
- [ ] Default to metadata-only synchronization with an explicit field allowlist.
- [ ] Add client-side redaction and size limits before events leave the machine.
- [ ] Encrypt transport and sensitive database columns; use managed secret/key storage in production.
- [ ] Implement strict tenant isolation tests at repository, service, API, job, cache, and object-storage layers.
- [ ] Use signed role/policy/approval bundles and protect against replay/downgrade.
- [ ] Add immutable tenant audit events with integrity checks and configurable retention.
- [ ] Add account/project deletion with documented backup-retention behavior.
- [ ] Add dependency, container, infrastructure, secret, and SAST scanning.
- [ ] Prepare for SOC 2 evidence collection without claiming compliance before an audit.

## Billing and packaging hypothesis

Use developer seats as the primary SaaS value metric. Bill optional managed model/compute usage separately.

| Plan | Initial price hypothesis | Intended user | Core package |
|---|---:|---|---|
| Community | Free | Local individuals and open source | Full local CLI/runtime, local roles/projects/history |
| Pro | $19/month or $190/year | Individual power users | Cloud sync, private role library, up to five managed projects, dashboard, extended history |
| Team | $39/user/month or $390/year | Engineering teams | Shared roles, projects, policies, approvals, analytics, GitHub App, audit history |
| Enterprise | Starting around $500/month; custom | Larger organizations | SSO/SCIM, advanced access, retention, audit export, private networking, support |
| Managed execution | Usage based | Customers requesting cloud workers | Compute/model charges isolated from subscriptions |

- [ ] Treat these prices as hypotheses and validate them with design partners before hard-coding plan limits.
- [ ] Implement entitlements centrally in `cloud/internal/billing/` and expose a typed capability response to the dashboard.
- [ ] Mirror enough entitlement state locally for graceful offline use of already-synced configuration.
- [ ] Never disable core local CLI functionality because a cloud subscription expires.
- [ ] Support monthly/annual billing, trials, plan changes, cancellation, grace periods, failed payments, invoices, and tax-aware checkout through Stripe.
- [ ] Add usage/limit warnings before blocking a SaaS capability.

## Delivery plan

### Phase 0 — Customer validation

- [ ] Interview at least ten engineering leads actively using two or more coding agents.
- [ ] Recruit five design partners with at least two repositories each.
- [ ] Validate whether shared roles, approvals, portfolio visibility, policies, or analytics drives the strongest willingness to pay.
- [ ] Pre-sell a limited design-partner plan before building remote execution.
- [ ] Define privacy fields customers will and will not allow off-device.

### Phase 1 — Foundation

- [ ] Canonical contracts and golden fixtures
- [ ] Accounts, organizations, teams, membership, and devices
- [ ] `dacli login` and signed device registration
- [ ] Project registration and metadata-only sync
- [ ] Database migrations, audit events, and tenant isolation tests
- [ ] Basic organization/project dashboard

### Phase 2 — Paid collaboration MVP

- [ ] Versioned shared role registry
- [ ] Role sync, pinning, diff, upgrade, rollback, and signature verification
- [ ] Cross-project run/event summaries
- [ ] Approval inbox and local approval continuation
- [ ] Organization/project policies and budget limits
- [ ] GitHub App installation, webhooks, and PR/check status
- [ ] Stripe subscriptions and plan entitlements

### Phase 3 — Intelligence and enterprise readiness

- [ ] Outcome/cost/reliability analytics
- [ ] Role/runtime comparisons
- [ ] Policy templates and staged releases
- [ ] Audit/analytics export and retention controls
- [ ] SAML/OIDC, SCIM, service accounts, and advanced access controls
- [ ] Slack/email notifications and approval integrations

### Phase 4 — Optional managed execution

- [ ] Remote-runner protocol and identity
- [ ] Customer-managed runner registration
- [ ] Ephemeral managed sandbox proof of concept
- [ ] Network/filesystem/tool policies enforced at the runner
- [ ] Usage metering, cost controls, artifact handling, and secure teardown

Do not begin Phase 4 until customers repeatedly use and pay for Phases 1–3.

## Success metrics

- [ ] At least five design partners connect multiple repositories.
- [ ] At least three design partners become paying customers.
- [ ] Weekly returning manager/reviewer usage demonstrates dashboard value.
- [ ] Teams reuse released role versions across multiple projects.
- [ ] Approval requests are resolved through the SaaS rather than side channels.
- [ ] Customers can identify failed/stale/over-budget work faster than through GitHub and terminals alone.
- [ ] Metadata-only operation passes security review for initial customers.
- [ ] SaaS gross margin remains predictable because model/compute costs are not hidden inside flat subscriptions.

## MVP acceptance criteria

- [ ] A user can create an organization, invite a teammate, register a device, and connect two repositories.
- [ ] A role can be created, tested, released, pinned to projects, synchronized, verified locally, upgraded, and rolled back.
- [ ] Local dacli runs appear in the cross-project dashboard without source code or secret values leaving the machine.
- [ ] A central policy can pause a local action and generate an approval request; an authorized decision resumes only the exact approved action.
- [ ] GitHub PR/check state is linked and reconciled through the GitHub App.
- [ ] Budget and outcome analytics are derived from documented, privacy-filtered events.
- [ ] Tenant isolation, offline replay, duplicate events, revoked devices, expired policies, and webhook redelivery have automated tests.
- [ ] A paid Team subscription activates shared roles, policies, approvals, analytics, and audit-history entitlements.
- [ ] The local CLI remains fully usable without a SaaS subscription.

## Open product decisions

- [ ] Should the SaaS implementation live in this repository, a private monorepo, or a separately licensed repository?
- [ ] Which event fields are enabled by default versus explicit opt-in?
- [ ] Is the first paying persona an individual power user, engineering manager, platform team, or AI enablement team?
- [ ] Are active developer seats, managed projects, or a hybrid the best value metric after design-partner interviews?
- [ ] Which role/policy fields may projects override locally?
- [ ] Which approvals require separation of duties?
- [ ] How long should run metadata and audit evidence be retained per plan?
- [ ] Is self-hosted control plane required before enterprise launch?

## Acceptance
## Log
- 2026-08-13T16:06:26Z blocked: Roadmap epic, not one-sitting executable work; continue through task-sized children beginning with 425
