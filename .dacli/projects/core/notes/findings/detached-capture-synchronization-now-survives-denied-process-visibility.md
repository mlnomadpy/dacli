---
id: f-detached-capture-synchronization-now-survives-denied-process-visibility
kind: note
note_kind: finding
created: 2026-08-12T16:38:18Z
created_by: a-codex-maintainer-s5kkg3
about: "[[384]]"
severity: major
---
# Detached capture synchronization now survives denied process visibility
internal/features/execution/execruntime_test.go waits for recorder complete after stdin capture; its injected-unobservable regression fails under the old early-return mutation with 'unobservable ProcState was mistaken for exit after 17.458µs', while the corrected focused execution/procmon suites pass in the denied-ps sandbox.
