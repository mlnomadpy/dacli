---
id: d-fail-closed-for-every-remote-operation-in-effective-pr-mode
kind: note
note_kind: decision
created: 2026-08-14T00:24:18Z
created_by: a-maintainer-2vktb5
about: "[[449]]"
---
# Fail closed for every remote operation in effective PR mode
## Chose
Fail closed for every remote operation in effective PR mode
## Rejected
Preserve the legacy network-error fallback to a local merge
## Because
A project PR policy exists to require GitHub checks and reviews; an outage cannot safely authorize bypassing those gates, and failed remote operations must leave the task recoverable and unlanded.
