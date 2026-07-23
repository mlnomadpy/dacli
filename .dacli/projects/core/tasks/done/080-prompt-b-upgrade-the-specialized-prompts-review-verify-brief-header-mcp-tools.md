---
id: t-01KY5X8BKXW1DWMRDQWF3RQEPV
kind: task
created: 2026-07-22T21:56:06Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# PROMPT-B: upgrade the specialized prompts (review, verify, brief-header, mcp-tools) — newest features + better framing
## Acceptance
- [x] mcp_tools.md documents the CURRENT command surface for MCP clients: accept, ship (--pr), integrate, wait, agents (--tail), kill, logs, github push/pull/project, catalog, calibrate, spawn --advise/--claim/--detach/--max-tokens — no missing or stale tools
- [x] review_workflow.md reflects the real review flow (gh pr diff, verify verdicts, trust-floor, --request-changes); verify_refute.md sharpens the adversarial-refute framing; brief_header.md keeps its emphasized SYSTEM data-not-instructions line
- [x] better framing throughout; accurate to shipped behavior
- [x] committed by an agent and opened as a PR; go build + go test ./internal/... green
## Log
- 2026-07-22T21:56:23Z claimed by a-gx269dxyzs
- 2026-07-22T22:03:24Z accepted by a-root
- 2026-07-22T22:03:24Z completed by a-root
- 2026-07-22T23:52:35Z a-gx269dxyzs: PR opened: https://github.com/mlnomadpy/dacli/pull/40 (event 01KY5XGZR4CJAR0HJC4JCQNMGC)
