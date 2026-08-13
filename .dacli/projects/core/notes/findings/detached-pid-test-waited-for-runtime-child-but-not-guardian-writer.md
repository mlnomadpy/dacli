---
id: f-detached-pid-test-waited-for-runtime-child-but-not-guardian-writer
kind: note
note_kind: finding
created: 2026-08-13T15:31:41Z
created_by: a-fixer-j57wh6
about: "[[417]]"
severity: major
---
# detached PID test waited for runtime child but not guardian writer
internal/features/execution/execruntime_test.go:62 now waits after the recorder completion marker until the reported guardian PID is no longer observable; the guardian can outlive its runtime child while writing runtime-exit.txt beside the TempDir transcript. Mutation: restoring the old immediate return made TestDetachedCompletionWaitsForGuardianAfterRecorder fail at execruntime_test.go:441 with completion returned after 35.333µs while the guardian could still write into TempDir.
