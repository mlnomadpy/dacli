---
id: t-01KZYRZPSGFQRPAWRYM6NACDCC
kind: task
created: 2026-08-13T23:58:11Z
created_by: a-root
owner: a-root
depends_on: [450, 449]
github:
  issue: 656
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Apply PR landing policy to dacli loop recovery and documentation
## Context
Adopted from GitHub issue #656.

## Parent and dependency

Part of #637. Depends on the policy foundation and integrate/ship enforcement slices created alongside this issue.

## Scope

Make `dacli loop` preserve and obey the resolved landing policy across cycles, then document configuration and recovery. Do not duplicate the landing implementation owned by integrate/ship.

Implementation and claim boundary: `internal/features/orchestration` owns loop policy propagation, checkpoint recovery, and bounded-loop tests; `docs/GITHUB.md` and `docs/SELFHOSTING.md` own operator configuration and recovery; `skills/dacli` owns the portable agent playbook. Consume tasks 450 and 449 without changing their foundation or landing feature layers.

## Acceptance criteria

- [ ] Loop planning reads the shared effective landing policy and passes it through to the existing landing boundary.
- [ ] Loop dry-run names the effective mode, base, override state, PR action, and required gates.
- [ ] Restarting after push, PR creation, pending checks, or merge reuses the canonical branch and existing PR without duplicate events.
- [ ] A local-only fallback in `pr` mode is refused and leaves the task open or explicitly blocked with a recoverable reason.
- [ ] A bounded-loop integration test covers interruption and restart at each persisted landing checkpoint.
- [ ] User documentation explains configuration, precedence, GitHub authentication, required checks/reviews, explicit override auditing, and recovery commands.
- [ ] The dacli skill/reference documentation describes when operators should select PR landing for GitHub-first collaboration.
- [ ] Focused package tests and `go test ./...` pass.

## Acceptance
- [x] Loop planning reads the shared effective landing policy and passes it through to the existing landing boundary.
- [x] Loop dry-run names the effective mode, base, override state, PR action, and required gates.
- [x] Restarting after push, PR creation, pending checks, or merge reuses the canonical branch and existing PR without duplicate events.
- [x] A local-only fallback in `pr` mode is refused and leaves the task open or explicitly blocked with a recoverable reason.
- [x] A bounded-loop integration test covers interruption and restart at each persisted landing checkpoint.
- [x] User documentation explains configuration, precedence, GitHub authentication, required checks/reviews, explicit override auditing, and recovery commands.
- [x] The dacli skill/reference documentation describes when operators should select PR landing for GitHub-first collaboration.
- [x] Focused package tests and `go test ./...` pass.
## Log
- 2026-08-14T00:56:38Z claimed by a-maintainer-0c0w8g
- 2026-08-14T01:15:30Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/662 (event 01KZYX176CAM3TJ0C4C14MX7RM)
- 2026-08-14T01:17:03Z accepted by a-root
- 2026-08-14T01:17:03Z verified by `GOCACHE=/tmp/dacli-accept-448 go test ./...` (exit 0) in branch main at 3ab6530 — proves that tree builds, not that the work is in trunk
- 2026-08-14T01:17:03Z deliverable: dacli/448-apply-pr-landing-policy-to-dacli-loop-recovery-and-documentation is merged into main
- 2026-08-14T01:17:03Z completed by a-root
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-448 go test ./...","exit_code":0,"duration_ms":71844,"artifact_hash":"sha256:d8532bbd0bda9e27b5f219033d9cdb11210eb27535f726452554ad9d628dc8a5","verifier":"a-root","branch":"main","commit_sha":"3ab6530159a4ef52826d816188574c3d07100a20"}
