---
id: d-persist-stack-matched-project-roles-and-recorded-build-test-commands-in-the
kind: note
note_kind: decision
created: 2026-08-27T23:16:30Z
created_by: a-maintainer-mzxznz
about: "[[t-01M1068MNDZ72H5R35YRZ9MASK]]"
---
# Persist stack-matched project roles and recorded build/test commands in the operating profile and forward the roles to loop
## Chose
Persist stack-matched project roles and recorded build/test commands in the operating profile and forward the roles to loop
## Rejected
Let start preview and loop independently infer roles and derive verification from language defaults
## Because
One durable policy must govern dry-run and execution; independent inference caused issue #798's Android-to-Go divergence and could spawn before configuration errors surfaced.
