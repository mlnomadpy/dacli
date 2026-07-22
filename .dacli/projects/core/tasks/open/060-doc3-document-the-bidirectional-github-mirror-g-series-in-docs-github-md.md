---
id: t-01KY5G3H3XN8B0V9SJXT1RTP7D
kind: task
created: 2026-07-22T18:06:16Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 3, pessimistic: 5}
---
# DOC3: document the bidirectional GitHub mirror (G-series) in docs/GITHUB.md
## Acceptance
- [ ] docs/GITHUB.md documents github push (tasks->issues with status labels + close-on-done + backlink, decisions->labeled issues, findings->issue comments), github pull (inbound issues->tasks), github sync, and the pr enrichment + verify-verdicts-as-review-comments — verified against internal/features/ghmirror and vcs
- [ ] the disclosure gate is documented (github link --allow-public consent + live visibility re-check; every push operator-triggered) so a reader knows nothing publishes automatically
- [ ] committed by an agent; docs-only
## Log
- 2026-07-22T18:06:28Z claimed by a-36j29f5fcw
