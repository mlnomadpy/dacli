---
id: t-01KY3EKR1MSTD09QSJGSW6RSTM
kind: task
created: 2026-07-21T23:01:42Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 3, pessimistic: 5}
---
# Audit the Go codebase for performance and best-practice issues
## So that
the tool that ships zero-dep quality is itself high-quality Go
## Acceptance
- [x] findings filed for real perf or idiom issues (or a clean bill), each citing file:line
## Log
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: FindTask reads+parses the entire task tree per call; amplified to O(events×tasks) inside sync/taint/replay loops
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: brief.trim() re-renders the whole brief on every dropped section — O(k × total content)
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: Single-item Load* helpers read the entire directory off disk (LoadRole/LoadRuntime/LoadShortcut → LoadAll*)
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: git/gh subprocesses spawn with no context/timeout — a hung child blocks the whole MCP stdio server
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: Embedded, immutable templates re-read+re-parsed on every call (prompts.MCPDesc, gates.Get) and skill dirs scanned twice per load
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: collab.cmdThreads reads each question file a 3rd time after eventlog.List already parsed 'applied', and walks the event tree twice
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: replay reads run metadata (2 file opens) for every run dir even in single-prefix mode; FindTask hoistable out of the loop
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: eventlog.apply is non-atomic: a mid-apply failure leaves the event pending and re-runs it, duplicating notes / log lines on next sync
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: gitx.Merge reports every merge failure as a conflict, discarding the real error — a non-conflict failure wrongly blocks the task
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: Two avoidable algorithmic inefficiencies in spm: kahn re-sorts the ready frontier every pop; maskCode copies the whole tail per code fence
- 2026-07-21T23:09:25Z finding by a-hp8fwzbck0: Swallowed I/O errors: ReadDir failures reported as empty, run-record writes discarded, WriteFile orphans its temp file on rename failure
- 2026-07-22T15:29:28Z accepted by a-root
- 2026-07-22T15:29:28Z completed by a-root
