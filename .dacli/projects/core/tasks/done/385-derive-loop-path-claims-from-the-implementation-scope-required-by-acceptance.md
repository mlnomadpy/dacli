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
- [x] A loop task whose acceptance names docs/RUNTIMES.md plus runtime persistence, execution, and CLI behavior receives claims covering docs/RUNTIMES.md, internal/store, internal/features/execution, and internal/cli before spawn
- [x] The spawned implementer can commit every file required by that inferred scope while dacli commit still refuses an unrelated path
- [x] A regression fixture reproduces task 371, where documentation was the only literal path and the prior claim omitted six required code files
- [x] go test -race ./internal/features/orchestration ./internal/features/execution ./internal/store ./internal/cli passes
## Log
- 2026-08-12T16:46:32Z claimed by a-codex-maintainer-nmzkpw
- 2026-08-12T16:55:30Z accepted by a-root
- 2026-08-12T16:55:30Z closed WITHOUT verification — no --verify command was given
- 2026-08-12T16:55:30Z deliverable: dacli/385-derive-loop-path-claims-from-the-implementation-scope-required-by-acceptance exists but is NOT in main — closed anyway
- 2026-08-12T16:55:30Z completed by a-root
