---
id: d-resume-root-invoked-supervise-only-from-a-matching-root-recovery-transfer
kind: note
note_kind: decision
created: 2026-08-28T15:14:32Z
created_by: a-fixer-wskwrc
about: "[[t-01M14ABSH137KVYGMWBY9FC927]]"
---
# Resume root-invoked supervise only from a matching root recovery transfer
## Chose
Resume root-invoked supervise only from a matching root recovery transfer
## Rejected
Redirect every main-checkout supervise invocation to any task branch worktree
## Because
The durable transfer binds root recovery to the registered canonical branch while preserving ordinary main-checkout behavior and refusal semantics.
