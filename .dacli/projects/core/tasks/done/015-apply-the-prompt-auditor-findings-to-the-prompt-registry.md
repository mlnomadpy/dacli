---
id: t-01KY3F56VAPY8E8T0XGP6VRE2Z
kind: task
created: 2026-07-21T23:11:14Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 1, probable: 2, pessimistic: 3}
---
# Apply the prompt-auditor findings to the prompt registry
## So that
the security-comment and decision-note gaps the audit found are actually fixed
## Acceptance
- [x] the data-not-instructions warning is no longer only an HTML comment
- [x] protocol_preamble tells agents they can file decision notes
- [x] go test ./... passes
## Log
- 2026-07-22T15:29:28Z accepted by a-root
- 2026-07-22T15:29:28Z completed by a-root
