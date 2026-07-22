---
id: t-01KY5HCHN5R1YQ3FA6K9EH8CK2
kind: task
created: 2026-07-22T18:28:40Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# PROVISION: adopt --provision-roles seeds a team-provisioning task for a role-architect

## Context
Today `dacli adopt` (internal/features/onboard/onboard.go, `cmdAdopt` :48) onboards a repo: writes the Codebase map with a **Languages** breakdown (:217) and optionally seeds TODO tasks. The operator wants adopt to also provision the TEAM — an agent decides which roles the project needs, fetches their skills, and creates them — BEFORE work starts.

Build a `--provision-roles` flag on `cmdAdopt`. After the normal onboarding, when set, seed ONE task via `store.CreateTask(w, id, projectSlug, "Provision the team for <project>", opts)` whose body (set the task's Context/body section) carries:
- the detected **Languages** and the **Codebase map** summary (you already computed these — reuse `mapBody`/the scan),
- a directive for a `role-architect` agent: "Analyze this project's stack and domains. Decide the MINIMAL role roster it needs (e.g. an implementer, a reviewer, a language-specific auditor, a docs writer — justify EACH against the codebase; do not over-staff). For each role: pick relevant skills from skills.sh and run `dacli skill fetch <owner/repo>`, then create the role with `dacli role add <name> --kind implementer|reviewer|researcher|designer --grant ro|rw --model <tier> --skills <...>`. Finish with `dacli note add decision` documenting the roster and why."
- Keep it slice-clean: onboard does NOT import execution — it SEEDS the task and PRINTS the next step (`next: dacli spawn --task <seq> --role role-architect`); the operator (or `dacli next`) spawns the architect. The `role-architect` role already exists in the workspace.

Do NOT auto-spawn from adopt (no cross-slice import; the operator triggers the architect). This is discovery-then-provision, matching the phase discipline: no implementation before the team is set.

## Scope (STRICT) — touch ONLY:
- `internal/features/onboard/onboard.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY onboard.go plus this task's file. `go build ./...` + `go test ./internal/...` green (onboard has tests — the default adopt path must be unchanged when --provision-roles is absent). `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [x] dacli adopt gains --provision-roles: after onboarding, it seeds a 'Provision the team for <project>' task whose brief carries the codebase map + detected languages + a directive to analyze the stack, decide the minimal role roster (justify each), skill-fetch from skills.sh per role, and role-add them
- [x] the seeded brief tells the architect the exact primitives: dacli skill fetch <owner/repo>, dacli role add <name> --kind <k> --grant <g> --model <m> --skills <...>, and to file a decision documenting the roster; adopt prints 'next: dacli spawn --task <n> --role role-architect'
- [x] committed by an agent; go build + go test ./internal/... green; onboard slice change only, no cross-slice import
## Log
- 2026-07-22T18:29:15Z claimed by a-xkktk9s4kk
- 2026-07-22T18:36:54Z accepted by a-root
- 2026-07-22T18:36:54Z completed by a-root
