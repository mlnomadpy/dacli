---
id: d-wait-for-both-recorder-completion-and-guardian-exit-in-the-shared-detached-test
kind: note
note_kind: decision
created: 2026-08-13T15:31:41Z
created_by: a-fixer-j57wh6
about: "[[417]]"
---
# wait for both recorder completion and guardian exit in the shared detached-test helper
## Chose
wait for both recorder completion and guardian exit in the shared detached-test helper
## Rejected
wait only for the recorder completion marker or weaken the PID assertion
## Because
the recorder marker settles capture writes but the reported PID is the guardian, which persists runtime-exit.txt after its child exits; the helper covers every detached test without changing production detach behavior
