---
id: t-01M11HZW2DMC87Z8TNWGX7DFQ1
kind: task
created: 2026-08-27T12:09:21Z
created_by: a-root
owner: a-root
github:
  issue: 817
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Define dacli as the control plane for autonomous coding-agent swarms
## Context
Adopted from GitHub issue #817.

## Problem

The current product story mixes a human-operated CLI, an autonomous engineering team, and an architecture that says dacli is never the conductor. The intended user is an orchestrator AI agent that uses dacli to govern coding-agent CLIs across the full product-building lifecycle. This actor model and the boundary between agent intelligence and dacli policy/orchestration are not stated consistently.

## Design direction

Define the system as an agent-facing control plane and bounded orchestration runtime. Humans provide product direction and authority; an orchestrator agent plans and judges; dacli provides durable state, policy, routing, lifecycle, budgets, recovery, and evidence; coding-agent CLIs implement/review/test; GitHub carries visible collaboration and landing. Preserve the distinction between deterministic orchestration and model judgment.

Resolve the contradiction between README claims that the loop runs the software process and docs/ARCHITECTURE.md saying nothing in dacli walks a project to completion.

## Acceptance
- [x] README.md, DESIGN.md, docs/ARCHITECTURE.md, and docs/PROPOSALS.md use one consistent agent-first actor model.
- [x] The docs distinguish human authority, orchestrator-agent judgment, dacli policy/orchestration, coding-CLI execution, and GitHub collaboration.
- [x] Autonomous operation is described as governed, bounded, recoverable execution rather than guaranteed product judgment.
- [x] Codex, Claude Code, Gemini CLI, Copilot CLI, and generic adapters are presented symmetrically.
- [x] Normative architecture no longer contradicts the shipped loop behavior.
- [x] Documentation links and repository documentation checks pass.
## Log
- 2026-08-27T12:21:42Z accepted by a-root
- 2026-08-27T12:21:42Z verified by `go test ./...` (exit 0) in branch main at fefc6f7 — proves that tree builds, not that the work is in trunk
- 2026-08-27T12:21:42Z deliverable: dacli/518-define-dacli-as-the-control-plane-for-autonomous-coding-agent-swarms is merged into main
- 2026-08-27T12:21:42Z completed by a-root
- 2026-08-27T12:45:17Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/821 (event 01M11J7Q8YV5HRAMYKATVX77GE)
## Verification Evidence
{"command":"go test ./...","exit_code":0,"duration_ms":56727,"artifact_hash":"sha256:e000606d290badec0c686923c724244d25683508ccae0a4a591d690f9499f57e","verifier":"a-root","branch":"main","commit_sha":"fefc6f7137d4a798b4f244bea43d7bfa09f9cd10"}
