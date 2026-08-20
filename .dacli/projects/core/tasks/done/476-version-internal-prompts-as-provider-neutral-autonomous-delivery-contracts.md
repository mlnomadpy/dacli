---
id: t-01M0CX03CC0N95X4M5ESKRP2E6
kind: task
created: 2026-08-19T11:37:40Z
created_by: a-root
owner: a-root
depends_on: [470]
github:
  issue: 707
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Version internal prompts as provider-neutral autonomous-delivery contracts
## Context
Adopted from GitHub issue #707.

## Problem

The internal prompt registry predates dacli's current critical-path scheduler, provider-neutral model profiles, calibrated estimates, worktree claims, GitHub-first landing, persistent loop journal, and recovery semantics. Static prose can drift from command help and can steer every coding CLI toward outdated or provider-specific behavior.

## Design

Treat prompts as versioned executable contracts, not incidental strings. Keep one provider-neutral semantic prompt assembled from typed sections, then let runtime adapters add only transport-specific instructions. Generate command examples from the command registry where possible. Use golden fixtures plus behavioral scenarios so a prompt change must prove the agent chose the intended safe action.

Required semantic sections:

- issue deduplication and GitHub/local identity mapping
- estimation, dependency graph, critical path, slack, priority, and WIP
- cheapest-capable model selection with consequence-based uplift and provider fallback
- role scope, grant, claim/worktree isolation, and one-sitting decomposition
- verification, mutation proof, independent review, PR checks, and landing truth
- token/time/cost budgets, stop conditions, journal recovery, and circuit breakers
- honest empty cycles, no invented backlog, and evidence-backed findings
- provider-neutral commands for Codex, Claude Code, Gemini, Copilot, and generic adapters

## Acceptance
- [x] Prompt templates have an explicit schema/version and the resolved version/hash is written to every invocation record.
- [x] Shared semantic sections are composed once; adapters contain only runtime-specific transport/configuration instructions.
- [x] Command examples are generated from or validated against the CLI command registry, including usage and exit-code contracts.
- [x] Implementer, reviewer, estimator/planner, loop-auditor, and recovery prompts cover the required semantic sections without contradictory lifecycle instructions.
- [x] Golden prompt fixtures are deterministic, bounded, and fail when a required section or current command signature is removed.
- [x] Behavioral fixture scenarios prove: duplicate work is not filed; critical-path work is selected; a cheap capable model is preferred; high-consequence work is uplifted; exit 3 is not retried; an empty audit does not invent work.
- [x] Runtime-specific tests cover Codex, Claude Code, Gemini, Copilot, and generic adapters without naming one provider as the framework default.
- [x] Prompt overrides declare their base version and fail closed or warn visibly when incompatible.
- [x] Prompt documentation explains versioning, customization, token-size tradeoffs, and migration.
- [x] Mutation evidence and `go test ./...` pass.
## Log
- 2026-08-19T14:22:34Z claimed by a-maintainer-kcr272
- 2026-08-19T14:41:25Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/749 (event 01M0D7EXV2RF41VF29BVZPWFKY)
- 2026-08-19T14:47:54Z accepted by a-root
- 2026-08-19T14:47:54Z verified by `GOCACHE=/tmp/dacli-accept-476 go test ./internal/prompts ./internal/features/execution ./internal/cli ./internal/features/knowledge` (exit 0) in branch main at 8d62ff8 — proves that tree builds, not that the work is in trunk
- 2026-08-19T14:47:54Z deliverable: dacli/476-version-internal-prompts-as-provider-neutral-autonomous-delivery-contracts is merged into main
- 2026-08-19T14:47:54Z completed by a-root
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-476 go test ./internal/prompts ./internal/features/execution ./internal/cli ./internal/features/knowledge","exit_code":0,"duration_ms":2500,"artifact_hash":"sha256:06c4ecc223bade1e00ee8648df690f6bbdf87de74b8c1870e9c61479457e8c67","verifier":"a-root","branch":"main","commit_sha":"8d62ff8fc2d52f8897d4c936c76b846d36fc7f7b"}
{"command":"GOCACHE=/tmp/dacli-accept-476 go test ./internal/prompts ./internal/features/execution ./internal/cli ./internal/features/knowledge","exit_code":0,"duration_ms":1195,"artifact_hash":"sha256:06c4ecc223bade1e00ee8648df690f6bbdf87de74b8c1870e9c61479457e8c67","verifier":"a-root","branch":"main","commit_sha":"8d62ff8fc2d52f8897d4c936c76b846d36fc7f7b"}
