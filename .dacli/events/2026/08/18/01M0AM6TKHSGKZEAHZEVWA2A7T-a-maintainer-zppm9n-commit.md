---
id: 01M0AM6TKHSGKZEAHZEVWA2A7T
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-18T14:25:34Z
created_by: a-maintainer-zppm9n
about: "[[t-01M0AEG5AQPVJTH41MJNFRGSSX]]"
origin: agent
applied: true
checksum: sha256:cb0701cfad2ef54f0c6abc29afda3f33fac33381d13299497465396f81a0c1f5
---
f0aa6da t-01M0AEG5AQPVJTH41MJNFRGSSX: make run records atomic and fail closed

Unify execution artifacts behind one atomic writer so critical prompt,
invocation, process, and outcome failures cannot be reported as successful.
Persist optional-artifact failures for runs show instead of stderr alone.

Mutation: replacing the detached outcome critical write with best-effort made
TestFinalizeRunOutcomeFailureDoesNotCompleteProcessOrAppendExit fail at
runrecord_writer_test.go:71: terminal outcome failure must visibly fail finalization.
role: maintainer
