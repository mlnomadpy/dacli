---
id: t-01KZXN0N0E6JSXPPK7RTR0GBGC
kind: task
created: 2026-08-13T13:29:33Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 571
  repo: mlnomadpy/dacli
---
# Add a multi-CLI and model-routing operator guide
## Acceptance
- [ ] The guide covers all nine shipped presets and explains their read-only and read-write pairs.
- [ ] The guide includes setup, doctor, preflight, and spawn examples for Codex with concise equivalents for Claude, Gemini, Copilot, and generic-exec.
- [ ] The guide explains model selection, capacity and cost routing, provider limit classification, circuit breakers, and explicit fallback chains.
- [ ] The guide explains conserving provider limits, recommendation overrides, and troubleshooting via observable commands and exit codes.
- [ ] The guide is present in docs/README.md and mkdocs.yml, and documentation support tests plus go test ./... pass.
## Log
