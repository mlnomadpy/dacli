---
id: t-01M1493JFF0JAMR7ZW3G90DKBJ
kind: task
created: 2026-08-28T13:31:49Z
created_by: a-root
owner: a-root
github:
  issue: 876
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Preserve external command diagnostics in typed CLI errors
## Context
Adopted from GitHub issue #876.

## Parent

Extracted from #871 and reproduced on task 542 when `dacli commit` returned only `git add: exit status 128`, hiding the underlying Git failure.

## Objective

Preserve structured subprocess diagnostics across CLI boundaries without leaking secrets or dumping unbounded output.



## Non-goals

- Persisting unlimited command output.
- Exposing full environment variables.
- Replacing typed domain-level diagnoses such as `pr diagnose`.

## Manual workaround today

Operators rerun the underlying Git/GitHub command outside dacli to recover the diagnostic dacli discarded.

## Acceptance
- [ ] Every governed external command failure retains executable/operation, exit code or signal/timeout, bounded sanitized stdout/stderr tails, cwd scope, retryability, and suggested next action.
- [ ] Human output includes the most actionable underlying line; JSON exposes stable typed fields rather than parsing a formatted error string.
- [ ] Authentication tokens, environment secrets, credential-helper output, and paths outside the disclosed workspace are redacted by one shared policy.
- [ ] Wrapped errors retain their typed cause through feature and CLI layers; callers can distinguish policy refusal, timeout, contention, missing binary, and command failure.
- [ ] Successful quiet mutations print a concise object/result identity where silence would make state ambiguous.
- [ ] Fixtures cover Git index-lock failure, GitHub auth failure, command timeout, signal termination, multiline stderr, and secret redaction.
- [ ] Mutation tests fail when the root cause is collapsed to bare `exit status N` or secret material reaches output.
## Log
