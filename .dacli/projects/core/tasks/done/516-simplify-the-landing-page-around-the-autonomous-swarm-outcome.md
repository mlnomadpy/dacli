---
id: t-01M11HZVVYBQ4YEA08PR6GY61A
kind: task
created: 2026-08-27T12:09:21Z
created_by: a-root
owner: a-root
github:
  issue: 819
  repo: mlnomadpy/dacli
depends_on: "[t-01M11HZW2DMC87Z8TNWGX7DFQ1]"
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Simplify the landing page around the autonomous swarm outcome
## Context
Adopted from GitHub issue #819.

## Problem

The landing page leads with implementation vocabulary and a broad command surface before establishing the agent-facing product, its actors, and the shortest path from product direction to verified GitHub delivery. This makes dacli look like a human task tracker or a collection of unrelated CLI features.

## Design direction

Make docs/index.md a concise outcome-first page for agents and the humans governing them: product direction enters, an orchestrator agent drives dacli, coding-agent CLIs execute in isolated work, and verified changes land through GitHub. Keep advanced primitives discoverable without teaching the whole architecture above the fold.

## Acceptance
- [x] docs/index.md leads with the agent-facing control-plane value proposition and names supported coding CLI families without provider favoritism.
- [x] The primary flow is visually and textually reduced to direction → plan/route → execute/review → verify/land → learn/repeat.
- [x] The page separates the human governor, orchestrator agent, dacli, coding-agent runtimes, and GitHub responsibilities.
- [x] The first runnable path uses the operating-profile/start workflow instead of requiring readers to assemble low-level commands.
- [x] Detailed capability and command material remains linked but does not dominate the landing page.
- [x] The static docs build and link checks pass.
## Log
- 2026-08-27T12:09:32Z dependency edit by a-root (event 01M11J06ZZTKTJN63SDKAWDXJR)
- 2026-08-27T12:32:10Z accepted by a-root
- 2026-08-27T12:32:10Z verified by `go test ./docs` (exit 0) in branch main at 0d3cd62 — proves that tree builds, not that the work is in trunk
- 2026-08-27T12:32:10Z deliverable: dacli/516-simplify-the-landing-page-around-the-autonomous-swarm-outcome is merged into main
- 2026-08-27T12:32:10Z completed by a-root
- 2026-08-27T12:45:18Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/822 (event 01M11JWQ0VAHG0V4ZSXK769X5E)
## Verification Evidence
{"command":"go test ./docs","exit_code":0,"duration_ms":68,"artifact_hash":"sha256:f4796ba07855189b7bb28c2f14a6290f878f2a1a7bdb4e3a6a5a93f908459903","verifier":"a-root","branch":"main","commit_sha":"0d3cd62e92fe53ff3cef8b680c52eb63b38ed993"}
