---
id: t-01M0CX031NDQ5PQ8VRX1PQNWXE
kind: task
created: 2026-08-19T11:37:40Z
created_by: a-root
owner: a-root
github:
  issue: 708
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Publish the canonical cost-aware critical-path loop playbook and dacli skill
## Context
Adopted from GitHub issue #708.

## Problem

The repository docs and `skills/dacli/SKILL.md` describe many commands, but an agent still lacks one current decision framework for choosing task vs wave vs loop, sizing and routing models, organizing GitHub issues around dependencies and critical path, recovering partial runs, and operating safely for long periods. This makes users copy old one-off command sequences and underuses dacli's strongest value: a governed optimizer that repeatedly selects, verifies, and lands the highest-value safe work.

## Design

Publish one canonical operator playbook and make the dacli skill a concise router into focused references. Use progressive disclosure: the top-level skill states the decision flow; direct references cover operating profiles, model economics, critical-path backlog design, GitHub landing, continuous operations, and recovery. All examples must be checked against current `--help` and tested from a clean fixture.

The playbook should define the product vision precisely: dacli is a policy-driven continuous engineering controller over replaceable coding CLIs. “Continuous” means repeated bounded transactions with durable checkpoints, not permissionless infinite execution.



## Relationship to existing roadmap

- #437 remains the release-quality umbrella.
- #446 remains the optional SaaS/control-plane vision.
- This issue documents the best use of the local open-source system today and the safe bridge to long-running operation.

## Acceptance
- [x] README and docs present a first-choice table for inspect, single task, supervised wave, bounded loop, and future service operation.
- [x] The playbook gives an end-to-end GitHub-first flow: pull/deduplicate issues, estimate, dependencies/critical path, assign cheapest capable models, preview, execute, verify, PR/check/merge, sync/close, retro, repeat.
- [x] Model guidance distinguishes capability tier, estimated complexity, context size, consequence uplift, provider quotas/health, and independent-review diversity.
- [x] Continuous-operation guidance specifies finite cycles, rolling budgets, WIP, idle backoff, leases/heartbeats, STOP, journal recovery, circuit breakers, dead letters, observability, and default-off release publication.
- [x] Project isolation, workspace-wide collaboration state, task references, cross-project dependencies, and GitHub mapping are explained without implying tasks leak between projects.
- [x] Codex, Claude Code, Gemini, Copilot, and generic runtime examples are symmetrical and provider-neutral.
- [x] `skills/dacli/SKILL.md` remains concise and points one level deep to focused references; the installed skill is regenerated from committed repository content.
- [x] Every documented command is validated against current help or a docs test; outdated flags fail CI.
- [x] The skill passes `quick_validate.py` and a fresh-agent forward test successfully completes a representative plan-to-PR scenario without hidden context.
- [x] Docs clearly separate shipped behavior, experimental behavior, and future service/SaaS/GitHub-App vision.
- [x] Direct PR guidance explicitly requires observed merged/trunk state before owner acceptance and GitHub issue closure, distinguishes that path from `ship` owning a wave accept-plus-integrate transaction, and states that no dedicated runtime-cooldown clear/expiry command is shipped.
## Log
- 2026-08-19T11:39:42Z claimed by a-fixer-x51vke
- 2026-08-19T13:21:33Z accepted by a-root
- 2026-08-19T13:21:33Z verified by `python3 /Users/tahabsn/.codex/skills/.system/skill-creator/scripts/quick_validate.py skills/dacli && diff -qr skills/dacli /Users/tahabsn/.codex/skills/dacli && GOCACHE=/tmp/dacli-gocache-475-final go test ./docs -run TestPublicSupportClaimsMatchShippedSurface` (exit 0) in branch main at 6f8ba91 — proves that tree builds, not that the work is in trunk
- 2026-08-19T13:21:33Z deliverable: dacli/475-publish-the-canonical-cost-aware-critical-path-loop-playbook-and-dacli-skill is merged into main
- 2026-08-19T13:21:33Z completed by a-root
- 2026-08-19T13:30:33Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/716 (event 01M0CYDT5FT7THG5RT9YV32G0T)
- 2026-08-19T13:30:33Z a-verifier-y9s74v: verify-verdict: no-verdict — codex-ro (a-verifier-y9s74v) on claim: From the committed skills/dacli/SKILL.md and its one-level references, with no prior transcript, a fresh operator can construct and validate a representative GitHub-first plan-to-PR flow including deduplication, estimation/dependencies, cheapest-capable routing, preview/spawn, monitoring/restart recovery, verification, project PR landing policy, branch push, conditional auto-merge, acceptance/ship ordering, retro, and bounded model-aware loop roles; exact commands agree with current help. — panelist reported nothing — counts as unconfirmed (event 01M0CZZ307HB2X1YE9YYMQJN66)
## Verification Evidence
{"command":"python3 /Users/tahabsn/.codex/skills/.system/skill-creator/scripts/quick_validate.py skills/dacli \u0026\u0026 diff -qr skills/dacli /Users/tahabsn/.codex/skills/dacli \u0026\u0026 GOCACHE=/tmp/dacli-gocache-475-final go test ./docs -run TestPublicSupportClaimsMatchShippedSurface","exit_code":0,"duration_ms":6525,"artifact_hash":"sha256:8f1cb822661abbf8c4647a26164a870e2f90e22fccd89e0ca8a7767d531f4305","verifier":"a-root","branch":"main","commit_sha":"6f8ba91360576659076803129165dee83983e4da"}
{"command":"python3 /Users/tahabsn/.codex/skills/.system/skill-creator/scripts/quick_validate.py skills/dacli \u0026\u0026 diff -qr skills/dacli /Users/tahabsn/.codex/skills/dacli \u0026\u0026 GOCACHE=/tmp/dacli-gocache-475-final go test ./docs -run TestPublicSupportClaimsMatchShippedSurface","exit_code":0,"duration_ms":343,"artifact_hash":"sha256:f2f5d671e7cc71ae8c6df02b020d94ef8ef77ccbfa5d7680236f6e3e4ecd6126","verifier":"a-root","branch":"main","commit_sha":"6f8ba91360576659076803129165dee83983e4da"}
