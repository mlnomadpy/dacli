---
id: t-01KZYRZPYGECMPYSEPDFS1QS3F
kind: task
created: 2026-08-13T23:58:11Z
created_by: a-root
owner: a-root
github:
  issue: 654
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Add typed project landing policy configuration and resolution
## Context
Adopted from GitHub issue #654.

## Parent

Part of #637.

## Scope

Add the domain/configuration foundation only. Persist a typed project landing policy and resolve command-line overrides into one effective value that downstream commands can consume. Do not implement GitHub landing behavior in this slice.

## Design

- `landing.mode`: enum `local` or `pr`
- `landing.base`: validated non-empty branch when configured
- legacy projects deserialize to `local` and preserve their existing base behavior
- one resolver owns config/default/CLI precedence and returns both the effective policy and whether an explicit override was used

## Acceptance criteria

- [ ] The project model persists and reloads `landing.mode` and `landing.base`.
- [ ] Unknown landing modes and an invalid configured base are rejected with exit code 2 before persistence.
- [ ] Existing project files without landing fields resolve to the legacy local-landing behavior.
- [ ] One shared resolver applies documented config/default/CLI precedence and reports an explicit override.
- [ ] Project inspection output and structured output expose the configured and effective landing values without leaking unrelated configuration.
- [ ] Table-driven tests cover round-trip persistence, legacy default, invalid values, and override precedence.
- [ ] The new regression tests fail when the typed policy fields or shared resolver are removed.
- [ ] Focused package tests and `go test ./...` pass.

## Acceptance
- [ ] The project model persists and reloads `landing.mode` and `landing.base`.
- [ ] Unknown landing modes and an invalid configured base are rejected with exit code 2 before persistence.
- [ ] Existing project files without landing fields resolve to the legacy local-landing behavior.
- [ ] One shared resolver applies documented config/default/CLI precedence and reports an explicit override.
- [ ] Project inspection output and structured output expose the configured and effective landing values without leaking unrelated configuration.
- [ ] Table-driven tests cover round-trip persistence, legacy default, invalid values, and override precedence.
- [ ] The new regression tests fail when the typed policy fields or shared resolver are removed.
- [ ] Focused package tests and `go test ./...` pass.
## Log
- 2026-08-13T23:59:13Z claimed by a-maintainer-ry3evs
