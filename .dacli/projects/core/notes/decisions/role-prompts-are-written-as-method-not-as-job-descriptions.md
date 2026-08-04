---
id: d-role-prompts-are-written-as-method-not-as-job-descriptions
kind: note
note_kind: decision
created: 2026-08-04T12:34:21Z
created_by: a-root
---
# Role prompts are written as method, not as job descriptions
## Chose
Role prompts are written as method, not as job descriptions
## Rejected
a one-paragraph summary per role
## Because
The prompt is the only thing standing between a role name and generic behaviour. A prompt that says 'you are the REVIEWER, review carefully' produces exactly what a bare role name produces. The ones that have worked here — fixer, reviewer, go-auditor, maintainer, junior, integrator — share a shape: what this role does that others do not, the specific method (red-green-refactor; invariant-over-example tests; verify the finding before filing it), the commands it actually runs, and an explicit stop condition saying when to escalate rather than push on. Junior's 'when to stop and escalate' is load-bearing precisely because it runs on the cheap model. Every remaining role gets that shape; a role that cannot be given a distinct method does not need to exist and should be deleted rather than padded.
