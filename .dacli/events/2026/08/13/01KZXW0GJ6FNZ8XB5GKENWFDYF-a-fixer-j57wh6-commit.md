---
id: 01KZXW0GJ6FNZ8XB5GKENWFDYF
kind: event
event_kind: commit
created: 2026-08-13T15:31:48Z
created_by: a-fixer-j57wh6
about: "[[t-01KZXQDX92DDCGBYKGCHPJ9426]]"
origin: agent
applied: true
---
44ee191 417: wait for detached guardian before TempDir cleanup

The recorder completion marker only settles the runtime child; wait for the reported guardian PID because it can still write runtime-exit.txt.

Mutation: TestDetachedCompletionWaitsForGuardianAfterRecorder failed with "completion returned after 35.333µs while the guardian could still write into TempDir" when the guardian wait was removed.
role: fixer
