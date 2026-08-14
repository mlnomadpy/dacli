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
- [ ] Loop planning reads the shared effective landing policy and passes it through to the existing landing boundary.
- [ ] Loop dry-run names the effective mode, base, override state, PR action, and required gates.
- [ ] Restarting after push, PR creation, pending checks, or merge reuses the canonical branch and existing PR without duplicate events.
- [ ] A local-only fallback in `pr` mode is refused and leaves the task open or explicitly blocked with a recoverable reason.
- [ ] A bounded-loop integration test covers interruption and restart at each persisted landing checkpoint.
- [ ] User documentation explains configuration, precedence, GitHub authentication, required checks/reviews, explicit override auditing, and recovery commands.
- [ ] The dacli skill/reference documentation describes when operators should select PR landing for GitHub-first collaboration.
- [ ] Focused package tests and `go test ./...` pass.
## Log
- 2026-08-14T00:56:38Z claimed by a-maintainer-0c0w8g
