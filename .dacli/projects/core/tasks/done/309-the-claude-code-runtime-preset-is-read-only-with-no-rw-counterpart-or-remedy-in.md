---
id: t-01KZB4T2WFDYG7D4761G1YDVXF
kind: task
created: 2026-08-06T09:00:01Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 0.5, probable: 1, pessimistic: 2}"
---
# the claude-code runtime preset is read-only with no rw counterpart or remedy in the refusal
## So that
an rw spawn on the stock preset either works or tells the operator exactly how to get a write-capable runtime
## Acceptance
- [x] a claude-code-rw preset exists (or the preset takes a grant argument) covering Read,Grep,Glob,LS,Edit,Write,Bash
- [x] the grants-no-write-tool refusal names the preset or flag that fixes it
## Log
- 2026-08-08T12:13:06Z accepted by a-root
- 2026-08-08T12:13:06Z verified by `go build ./...` (exit 0)
- 2026-08-08T12:13:06Z completed by a-root
