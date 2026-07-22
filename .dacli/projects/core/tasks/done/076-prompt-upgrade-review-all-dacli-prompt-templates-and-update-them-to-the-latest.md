---
id: t-01KY5JSH6JAWAWZJDXJ6TE0T18
kind: task
created: 2026-07-22T18:53:14Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# PROMPT-upgrade: review all dacli prompt templates and update them to the latest features
## Acceptance
- [x] every template in internal/prompts/tpl/*.md is reviewed and upgraded to reference the CURRENT command surface + workflow: accept/ship/integrate, spawn --advise/--claim/--detach/--max-tokens, wait, agents --tail, the token-calibration and trust/taint gates, github push — no stale command names or removed flows
- [x] protocol_preamble, git_workflow, review_workflow, supervise_correction, brief_header, and mcp_tools.md are accurate to the shipped behavior; prior prompt-auditor findings are incorporated
- [x] committed by an agent; go build + go test ./internal/... green (prompt tests in internal/prompts + internal/cli must pass)
## Log
- 2026-07-22T18:53:33Z claimed by a-zs77k4nm1x
- 2026-07-22T19:59:36Z accepted by a-root (applied 1 proposal(s))
- 2026-07-22T19:59:36Z completed by a-root
