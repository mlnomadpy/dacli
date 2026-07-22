---
id: t-01KY5YNJBQ4GC8XZ320ZAZNRH4
kind: task
created: 2026-07-22T22:20:47Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# I2: dacli ship --pr --auto — set PRs to auto-merge when CI passes (hands-off integration)
## Acceptance
- [x] dacli ship --pr --auto sets each done-task PR to auto-merge via 'gh pr merge --auto --merge --delete-branch' so GitHub merges it when CI goes green — the operator never waits on CI or merges by hand
- [x] without --auto, ship --pr merges only PRs whose checks already pass (gh pr checks), and reports any it left open for a red/pending check instead of blindly merging
- [x] committed by an agent and opened as a PR; go build + go test ./internal/... green; gofmt clean
## Log
- 2026-07-22T22:21:15Z claimed by a-8pkc6y4kp7
- 2026-07-22T22:32:13Z accepted by a-root
- 2026-07-22T22:32:13Z completed by a-root
