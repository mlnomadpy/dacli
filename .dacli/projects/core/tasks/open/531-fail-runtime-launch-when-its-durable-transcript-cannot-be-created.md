---
id: t-01M13S7VDH9ZN15AJAEYS5QFC4
kind: task
created: 2026-08-28T08:54:32Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 847
  repo: mlnomadpy/dacli
---
# Fail runtime launch when its durable transcript cannot be created
## So that
every reported agent launch has the durable transcript required for wait, usage, recovery, and audit truth
## Acceptance
- [ ] Foreground execRuntime returns a contextual transcript-creation error before starting the runtime when transcriptPath cannot be created.
- [ ] Detached execRuntime returns the same fail-closed error and records no live process identity.
- [ ] Focused regressions prove the runtime binary is never invoked on transcript creation failure.
- [ ] Mutation evidence and go test ./internal/features/execution pass.
## Log
