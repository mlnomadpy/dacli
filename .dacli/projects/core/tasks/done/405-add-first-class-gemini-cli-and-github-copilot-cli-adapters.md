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
- [x] runtime add provides Gemini CLI enforced-RO and workspace-write presets with explicit model flags and structured usage parsing
- [x] runtime add provides Copilot CLI enforced-RO and workspace-write presets with explicit model flags and least-privilege tool allowlists
- [x] Both adapters pass the shared conformance contract on Linux and macOS fixtures
- [x] runtime doctor detects incompatible installed versions or changed flags and refuses unsafe enforced-RO claims
- [x] Documentation names authentication requirements without storing provider credentials in .dacli
## Log
- 2026-08-13T10:13:52Z claimed by a-codex-maintainer-1d99qt
- 2026-08-13T10:41:45Z accepted by a-root
- 2026-08-13T10:41:45Z verified by `cd /Users/tahabsn/Documents/GitHub/dacli/.dacli/worktrees/core-405-add-first-class-gemini-cli-and-github-copilot-cli-adapters && GOCACHE=/private/tmp/dacli-owner-405 go test ./internal/features/execution ./internal/store ./docs` (exit 0) in branch main at 4f6be10 — proves that tree builds, not that the work is in trunk
- 2026-08-13T10:41:45Z completed by a-root
- 2026-08-13T10:47:41Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/560 (event 01KZXB122HW0YMJW8CYYX33DQ8)
