---
id: t-01KZX7PXH4VT1DC5A5TGK59E28
kind: task
created: 2026-08-13T09:37:03Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
github:
  issue: 548
  repo: mlnomadpy/dacli
---
# Build an executable coding-CLI conformance contract
## So that
every runtime adapter proves the lifecycle and safety behaviors dacli depends on
## Context
Use ports-and-adapters with contract tests: one shared behavioral suite, thin CLI adapters, and fake executables for deterministic CI. A runtime is first-class only when it passes the contract.
## Acceptance
- [x] One shared suite checks prompt transport, model selection, structured terminal result, usage capture, timeout and cancellation, and exit classification
- [x] The suite checks enforced RO refusal and workspace-write behavior without paid prompts or live credentials
- [x] Codex, Claude Code, Gemini CLI, Copilot CLI, and generic-exec fixtures run through the same contract in CI
- [x] runtime doctor reports contract capabilities and distinguishes declared, verified, failed, and unsupported states
- [x] docs publish a generated conformance matrix sourced from executable results
## Log
- 2026-08-13T09:44:20Z claimed by a-loop-bootstrap-maintaine-ey8tse
- 2026-08-13T10:10:48Z accepted by a-root
- 2026-08-13T10:10:48Z verified by `cd /Users/tahabsn/Documents/GitHub/dacli/.dacli/worktrees/core-404-build-an-executable-coding-cli-conformance-contract && GOCACHE=/private/tmp/dacli-owner-404 go test ./internal/features/execution ./docs` (exit 0) in branch main at f244e11 — proves that tree builds, not that the work is in trunk
- 2026-08-13T10:10:48Z completed by a-root
