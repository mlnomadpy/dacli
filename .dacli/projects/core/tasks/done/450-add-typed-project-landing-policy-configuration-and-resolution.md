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
- [x] The project model persists and reloads `landing.mode` and `landing.base`.
- [x] Unknown landing modes and an invalid configured base are rejected with exit code 2 before persistence.
- [x] Existing project files without landing fields resolve to the legacy local-landing behavior.
- [x] One shared resolver applies documented config/default/CLI precedence and reports an explicit override.
- [x] Project inspection output and structured output expose the configured and effective landing values without leaking unrelated configuration.
- [x] Table-driven tests cover round-trip persistence, legacy default, invalid values, and override precedence.
- [x] The new regression tests fail when the typed policy fields or shared resolver are removed.
- [x] Focused package tests and `go test ./...` pass.
## Log
- 2026-08-13T23:59:13Z claimed by a-maintainer-ry3evs
- 2026-08-14T00:17:10Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/658 (event 01KZYSNME9MFCKEYCPMNAC1J7G)
- 2026-08-14T00:18:30Z accepted by a-root
- 2026-08-14T00:18:30Z verified by `GOCACHE=/tmp/dacli-accept-450 go test ./...` (exit 0) in branch main at dadcf23 — proves that tree builds, not that the work is in trunk
- 2026-08-14T00:18:30Z deliverable: dacli/450-add-typed-project-landing-policy-configuration-and-resolution is merged into main
- 2026-08-14T00:18:30Z completed by a-root
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-450 go test ./...","exit_code":0,"duration_ms":71174,"artifact_hash":"sha256:0d2fe2377f40d6de9a3b31e551d1fe6b1b1047934c1d9ed181ea349b98014995","verifier":"a-root","branch":"main","commit_sha":"dadcf238ae52c9ebb9391ca1076ca5f46d5f3f78"}
