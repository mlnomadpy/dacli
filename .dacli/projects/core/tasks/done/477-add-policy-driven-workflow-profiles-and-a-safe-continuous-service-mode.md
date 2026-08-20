---
id: t-01M0CX03Q4A1BM8JD9YQBCNGV0
kind: task
created: 2026-08-19T11:37:40Z
created_by: a-root
owner: a-root
depends_on: [465, 469, 471, 472, 476]
github:
  issue: 706
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
---
# Add policy-driven workflow profiles and a safe continuous service mode
## Context
Adopted from GitHub issue #706.

## Problem

dacli exposes powerful primitives (`next`, `spawn`, `supervise`, `ship`, and `loop`) but the first operator decision is implicit: am I auditing, completing one task, running a supervised wave, or operating a continuously improving project? New users must assemble flags from several documents, and unattended operators can accidentally omit a budget, idle bound, review gate, or landing policy.

The desired future is a VPS-resident operator that keeps improving a project. That must not be one literally infinite process. It needs a durable, inspectable policy whose executions remain bounded, resumable, budgeted, and stoppable.

## Design

Introduce a provider-neutral **OperatingProfile** policy object and a `dacli start` entry point (with an equivalent non-interactive configuration command). Use the Strategy pattern for execution mode and keep scheduling, routing, landing, and release as orthogonal policy components rather than mode-specific conditionals.

Initial profiles:

1. `inspect` — read-only audit, no implementation or external mutation.
2. `task` — one selected task through claim, verification, PR, and landing.
3. `wave` — a bounded parallel ready-work window with disjoint claims.
4. `loop` — bounded review/plan/build/land/retro cycles with idle backoff.
5. `service` — a durable supervisor that repeatedly invokes bounded loops, using a lease, health checks, circuit breakers, and a stop file.

The resolved profile must be printed and persisted at project scope. CLI flags may override it for one invocation with provenance. Release automation is a separate, default-off policy requiring explicit channel, cadence, gates, and publication authority; choosing `service` must never imply tag or release publication.

## Policy fields

- scheduling: project, priorities, dependency/critical-path ordering, width/WIP
- routing: allowed runtimes/providers/models, cheapest-capable tier, consequence uplift, fallback order
- execution: profile, cycle/turn bounds, idle/backoff, lease and heartbeat
- budgets: per-task, per-cycle, rolling token/time/cost ceilings
- verification: mutation/required commands, independent review count and provider diversity
- landing: local vs PR, checks, reviews, auto-merge, protected branch
- release: disabled by default; optional channel/cadence/version/gates
- recovery: journal, dead-letter/circuit-breaker thresholds, stop mechanism



## Relationship to existing roadmap

- #437 validates release readiness and autonomous scenarios; this issue supplies the local operating-mode contract those scenarios select.
- #446 may distribute organization policies later; the local profile remains useful offline and is the enforcement source on the runner.

## Acceptance
- [x] `dacli start` interactively selects `inspect`, `task`, `wave`, `loop`, or `service`; `--profile` provides the same behavior headlessly.
- [x] The resolved OperatingProfile is persisted per project, shown before execution, and available as JSON without launching work.
- [x] `--dry-run` shows selected tasks, critical-path/slack ordering, role/runtime/model routing, budgets, claims, verification, landing, and release policy.
- [x] Every non-inspect profile has finite default task/cycle/rolling-budget bounds; service mode supervises repeated bounded invocations rather than running one unbounded loop.
- [x] STOP, lease loss, exhausted budget, repeated infrastructure failure, and an unknown landing state stop at a durable checkpoint with an actionable status.
- [x] Model selection uses the cheapest capable tier for estimated complexity, raises the tier for security/persistence/high-ambiguity consequences, and records the reason.
- [x] Release publication remains disabled unless a separate explicit policy authorizes it; no profile silently pushes tags or releases.
- [x] Existing direct commands remain compatible and are represented as profile strategies rather than duplicated orchestration paths.
- [x] CLI/MCP schemas, migration/default behavior, golden policy fixtures, mutation evidence, and `go test ./...` are covered.
## Log
- 2026-08-19T14:52:38Z claimed by a-maintainer-3necr2
- 2026-08-20T08:01:01Z accepted by a-root
- 2026-08-20T08:01:01Z verified by `GOCACHE=/tmp/dacli-task477-cache go test ./internal/features/orchestration -run 'TestOperatingProfile|TestStart|TestService' -count=1` (exit 0) in branch main at 7c6ca6c — proves that tree builds, not that the work is in trunk
- 2026-08-20T08:01:01Z deliverable: dacli/477-add-policy-driven-workflow-profiles-and-a-safe-continuous-service-mode is merged into main
- 2026-08-20T08:01:01Z completed by a-root
- 2026-08-20T08:03:10Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/752 (event 01M0F2J52TC05SGXHT2Y5AE3BA)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-task477-cache go test ./internal/features/orchestration ./internal/cli","exit_code":0,"duration_ms":39534,"artifact_hash":"sha256:3a732e54191bd60813d7ff6db0ce5143bd38ec9d45953a526bb8469dcd9ad8e8","verifier":"a-root","branch":"main","commit_sha":"7c6ca6cdbe4457e16109c3ac83204749a25fbcf0"}
{"command":"GOCACHE=/tmp/dacli-task477-cache go test ./internal/features/orchestration -run 'TestOperatingProfile|TestStart|TestService' -count=1","exit_code":0,"duration_ms":700,"artifact_hash":"sha256:a3e0d10b813ba61b75fa0e94442648712332304e85f7b475644a2d582e9a3023","verifier":"a-root","branch":"main","commit_sha":"7c6ca6cdbe4457e16109c3ac83204749a25fbcf0"}
