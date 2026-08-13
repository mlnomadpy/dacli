---
id: t-01KZX7PXM8TKXJQJXVWZBEMC0H
kind: task
created: 2026-08-13T09:37:03Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 10}"
depends_on: "[404:FS]"
github:
  issue: 549
  repo: mlnomadpy/dacli
---
# Add first-class Gemini CLI and GitHub Copilot CLI adapters
## So that
the shipped runtime roster covers more than Codex and Claude Code
## Context
Implement thin adapters over the conformance port. Gemini supports headless stream-json and model selection; Copilot supports programmatic prompts, explicit models, and granular tool permissions. Do not grant allow-all by default.
## Acceptance
- [ ] runtime add provides Gemini CLI enforced-RO and workspace-write presets with explicit model flags and structured usage parsing
- [ ] runtime add provides Copilot CLI enforced-RO and workspace-write presets with explicit model flags and least-privilege tool allowlists
- [ ] Both adapters pass the shared conformance contract on Linux and macOS fixtures
- [ ] runtime doctor detects incompatible installed versions or changed flags and refuses unsafe enforced-RO claims
- [ ] Documentation names authentication requirements without storing provider credentials in .dacli
## Log
