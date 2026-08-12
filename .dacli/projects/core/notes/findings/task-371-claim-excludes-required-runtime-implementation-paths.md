---
id: f-task-371-claim-excludes-required-runtime-implementation-paths
kind: note
note_kind: finding
created: 2026-08-12T16:03:31Z
created_by: a-codex-maintainer-vc0pbd
about: "[[371]]"
severity: major
---
# Task 371 claim excludes required runtime implementation paths
dacli commit refused internal/features/execution/*, internal/store/runtimefiles.go, and internal/cli/runtime_test.go as outside the claim; only docs/RUNTIMES.md is claimed, although every code acceptance criterion requires those runtime files. Implementation and focused verification are complete but cannot be committed without owner claim correction.
