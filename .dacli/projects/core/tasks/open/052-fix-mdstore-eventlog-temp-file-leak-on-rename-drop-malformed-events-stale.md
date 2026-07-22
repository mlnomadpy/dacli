---
id: t-01KY59YW15K8JFAP21C8BABQ6F
kind: task
created: 2026-07-22T16:18:52Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 1, probable: 2, pessimistic: 4}
---
# FIX mdstore/eventlog: temp-file leak on rename, drop malformed events, stale comment
## Acceptance
- [ ] mdstore.WriteFile removes its temp file when os.Rename fails (re-verify — reviewer says it still leaks)
- [ ] eventlog.List does not silently drop a malformed/unreadable event: it surfaces or logs the error instead of hiding it; sync.go's stale logOnce comment is corrected
- [ ] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T16:19:10Z claimed by a-hq6ebk9c35
