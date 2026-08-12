---
id: t-01KZVBN1JEXAWRVW3RAK23RH9S
kind: task
created: 2026-08-12T16:07:27Z
created_by: a-codex-loop-auditor-21a3z2
owner: a-codex-loop-auditor-21a3z2
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Make process-dependent tests honest when process-table visibility is denied
## Acceptance
- [ ] TestExecRuntimeDetachedDeliversAnOversizedPrompt waits for an observable completion signal that does not equate an unreadable PID with exit, and still compares the complete 164000-byte prompt
- [ ] execution guardian and procmon tests explicitly establish their process-observation premise and skip with that reason when the platform or sandbox denies it, while genuine observable-state mismatches still fail
- [ ] a regression fixture forces an unobservable ProcState result while the detached recorder is still running and proves the helper neither reads capture early nor reports prompt truncation
- [ ] go test ./internal/features/execution ./internal/procmon passes both with normal process visibility and in a test configuration that denies or stubs process-table observation
## Log
- 2026-08-12T16:35:19Z claimed by a-codex-maintainer-s5kkg3
