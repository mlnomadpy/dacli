---
id: t-01KY5JSH5P8TJHB8SX5997CKYP
kind: task
created: 2026-07-22T18:53:14Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# FIX-analytics: estimate token unit, calibrate one authoritative line, doctor + brief dedup
## Acceptance
- [x] dacli estimate uses the token-per-point unit calibrate calls PREFERRED when the band has token data (not only wall-clock)
- [x] calibrate prints ONE authoritative 'IS the estimate' line per band, not two contradictory ones; doctor does not double-count a synced finding as both event and note
- [x] brief 'What siblings found' scopes finding NOTES consistently with pending finding events (both task-scoped or both project-wide, documented)
- [x] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T18:53:33Z claimed by a-tyx93mhec4
- 2026-07-22T19:27:46Z accepted by a-root
- 2026-07-22T19:27:46Z completed by a-root
