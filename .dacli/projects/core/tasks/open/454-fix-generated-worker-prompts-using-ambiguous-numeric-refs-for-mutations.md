---
id: t-01KZYZR9B4312NVSTNS8NMJ1CE
kind: task
created: 2026-08-14T01:56:28Z
created_by: a-codex-loop-auditor-ejgvrk
owner: a-codex-loop-auditor-ejgvrk
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# Fix generated worker prompts using ambiguous numeric refs for mutations
## So that
In a workspace with alpha/001 and beta/001, generated task check, task done, note --about, commit --task, push --task, PR --task, accept, integrate, and merge commands address the spawned task by its stable ULID instead of failing with 'ref 001 is ambiguous'; execution.go currently formats t.Seq into Ref at lines 2116 and 2178, and the manual workaround is to replace every generated numeric ref with the task ULID.
## Acceptance
- [ ] internal/features/execution tests generate both rw and ro worker prompts for one of two tasks sharing sequence 001 across projects and assert every mutating command uses that task's full ULID, not bare 001
- [ ] internal/features/execution/execution.go supplies the stable task ULID to protocol_preamble.md and git_workflow.md while human-readable prompt text may still display the sequence and slug
- [ ] go test ./internal/features/execution ./internal/prompts passes, and a mutation restoring fmt.Sprintf("%03d", t.Seq) for either prompt construction path makes the new regression test fail
## Log
