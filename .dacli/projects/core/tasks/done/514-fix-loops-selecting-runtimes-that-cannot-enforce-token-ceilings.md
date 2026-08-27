---
id: t-01M11H870RTFMFNYV0ZZ92K6KZ
kind: task
created: 2026-08-27T11:56:26Z
created_by: a-root
owner: a-root
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
github:
  issue: 816
  repo: mlnomadpy/dacli
---
# Fix loops selecting runtimes that cannot enforce token ceilings
## Acceptance
- [x] Hard-ceiling loop/profile planning excludes runtimes whose adapters lack token-limit support before any spawn and reports each capability exclusion.
- [x] If no hard-capable implementation or review runtime exists, dry-run and real execution fail closed with a configuration remedy instead of producing a zero-work cycle.
- [x] An explicit persisted advisory-token policy is forwarded consistently to implementation and review spawns, warns that per-run enforcement is unavailable, and retains measured rolling-window, timeout, cycle, and idle limits.
- [x] Explicit implementer/runtime choices remain authoritative and never silently downgrade from hard to advisory accounting.
- [x] Provider-neutral fixtures cover Codex, Claude, Gemini, Copilot, and generic adapters without scheduler-side vendor branching.
- [x] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-27T13:05:36Z accepted by a-root
- 2026-08-27T13:05:36Z closed WITHOUT verification — no --verify command was given
- 2026-08-27T13:05:36Z deliverable: dacli/514-fix-loops-selecting-runtimes-that-cannot-enforce-token-ceilings is merged into main
- 2026-08-27T13:05:36Z completed by a-root
- 2026-08-27T13:07:22Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/827 (event 01M11MRBZXHJ9KJXDDF1YGQPKB)
