---
id: t-01KZV2X3X392WYCFJ0WQQR96H1
kind: task
created: 2026-08-12T13:34:34Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
depends_on: [365, 372, 374, 375, 378]
github:
  issue: 464
  repo: mlnomadpy/dacli
---
# Add first-class Codex CLI runtime presets and structured results
## So that
Codex works as a supported runtime without hand-authored adapters or unstable text transcripts
## Acceptance
- [ ] runtime add accepts Codex read-write and read-only presets whose argument ordering matches codex exec, including non-interactive approval, model selection, stdin delivery, and ephemeral operation
- [ ] The Codex adapter consumes JSONL events and records the final message, session identity, exit outcome, and token usage without applying the Claude stream-json parser
- [ ] runtime doctor verifies Codex read-only isolation through the local codex sandbox helper or an equivalent no-model behavioral probe on supported platforms
- [ ] A fake Codex fixture covers global-versus-exec flag ordering, stdin prompts, JSONL parsing, nonzero exits, and read-only probe refusal without network access
- [ ] docs/RUNTIMES.md documents Codex as shipped support and cites the minimum tested CLI version
- [ ] go test -race ./... passes
## Log
