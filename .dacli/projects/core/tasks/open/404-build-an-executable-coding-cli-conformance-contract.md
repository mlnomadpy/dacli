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
- [ ] One shared suite checks prompt transport, model selection, structured terminal result, usage capture, timeout and cancellation, and exit classification
- [ ] The suite checks enforced RO refusal and workspace-write behavior without paid prompts or live credentials
- [ ] Codex, Claude Code, Gemini CLI, Copilot CLI, and generic-exec fixtures run through the same contract in CI
- [ ] runtime doctor reports contract capabilities and distinguishes declared, verified, failed, and unsupported states
- [ ] docs publish a generated conformance matrix sourced from executable results
## Log
