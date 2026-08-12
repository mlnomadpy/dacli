---
id: t-01KZVBXKBB700Y5RBJPTVR2XW2
kind: task
created: 2026-08-12T16:12:07Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Derive loop path claims from the implementation scope required by acceptance criteria
## Acceptance
- [ ] A loop task whose acceptance names docs/RUNTIMES.md plus runtime persistence, execution, and CLI behavior receives claims covering docs/RUNTIMES.md, internal/store, internal/features/execution, and internal/cli before spawn
- [ ] The spawned implementer can commit every file required by that inferred scope while dacli commit still refuses an unrelated path
- [ ] A regression fixture reproduces task 371, where documentation was the only literal path and the prior claim omitted six required code files
- [ ] go test -race ./internal/features/orchestration ./internal/features/execution ./internal/store ./internal/cli passes
## Log
- 2026-08-12T16:46:32Z claimed by a-codex-maintainer-nmzkpw
