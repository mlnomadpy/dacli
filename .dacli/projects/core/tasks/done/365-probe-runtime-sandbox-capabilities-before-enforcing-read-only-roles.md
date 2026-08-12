---
id: t-01KZV16D6Z5YFXJEDWWM509K7R
kind: task
created: 2026-08-12T13:04:41Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
github:
  issue: 458
  repo: mlnomadpy/dacli
---
# Probe runtime sandbox capabilities before enforcing read-only roles
## So that
A declared sandbox flag cannot be mistaken for verified read-only isolation
## Acceptance
- [x] runtime doctor reports declared, verified, and unknown read-only capability as distinct observable states
- [x] Spawning an ro role refuses runtimes whose read-only behavior is unknown or failed unless the documented cooperative-policy path applies
- [x] Tests cover at least one verified adapter, one declaration-only adapter, and one failed probe without invoking a network service
- [x] docs/RUNTIMES.md describes the actual probe and refusal semantics
- [x] go test -race ./... passes
## Log
- Audit scope and claim hints: `internal/store/runtimefiles.go`,
  `internal/store/runtimefiles_test.go`,
  `internal/features/execution/execution.go`,
  `internal/features/execution/runrecord_test.go`, and `docs/RUNTIMES.md`.
  Keep any implementation inside these paths unless the task log explains why
  an additional shared contract file is necessary.
- 2026-08-12T13:11:23Z claimed by a-codex-maintainer-8ntneg
- 2026-08-12T13:38:22Z completed by a-root
