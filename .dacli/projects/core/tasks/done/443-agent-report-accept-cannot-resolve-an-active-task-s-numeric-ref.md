---
id: t-01KZYQ5EF1YQ05NRA8GW3N9PQM
kind: task
created: 2026-08-13T23:26:22Z
created_by: a-root
owner: a-root
depends_on: [445]
github:
  issue: 636
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# [agent-report] accept cannot resolve an active task's numeric ref
## Context
Adopted from GitHub issue #636.

From task worktree dacli/001-define-the-firebase-to-supabase-migration-contract-and-rollback-plan, after a provenance commit, ran dacli accept 001 --verify with an explicit worktree cd. It returned ref 001 is ambiguous, listing supabase/001 among others. The task brief prescribes dacli accept 001 and says not to retry refused operations, leaving no accepted task state.

Implementation and claim boundary: generated worker completion references in `internal/prompts`, task context/brief assembly in `internal/brief` and `internal/features/briefing`, and owner acceptance messaging in `internal/features/acceptance`. Update `skills/dacli/references/workspace-tasks-projects.md` only as needed to document stable ULID versus human shorthand. Do not alter the general task resolver or the project-qualified reference support landed by task 445.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] Generated task briefs and completion instructions use the task ULID for mutating commands such as `accept` and `task check`, never a workspace-ambiguous sequence alone.
- [x] An active task can be accepted from its isolated worktree using the generated ULID command.
- [x] Bare numeric references remain accepted when unique and return an ambiguity error without mutation when multiple projects share the sequence.
- [x] Ambiguity guidance names only reference forms that the resolver actually accepts; project-qualified guidance is covered with issue #628 / task 445.
- [x] Regression tests create two projects with the same sequence and prove the old generated numeric command fails while the fixed generated command resolves the assigned task.
- [x] Documentation distinguishes stable machine references (ULIDs) from human shorthand.
## Log
- 2026-08-14T01:22:52Z claimed by a-maintainer-addy71
- 2026-08-14T09:00:13Z accepted by a-root
- 2026-08-14T09:00:13Z verified by `GOCACHE=/tmp/dacli-accept-443 go test ./...` (exit 0) in branch main at 0c1f80a — proves that tree builds, not that the work is in trunk
- 2026-08-14T09:00:13Z deliverable: dacli/443-agent-report-accept-cannot-resolve-an-active-task-s-numeric-ref is merged into main
- 2026-08-14T09:00:13Z completed by a-root
- 2026-08-14T09:11:42Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/665 (event 01KZZMP472BWP0S8SW7H5W7307)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-443 go test ./...","exit_code":0,"duration_ms":61935,"artifact_hash":"sha256:3f2ed47f9803b324d89cfe38adea68cad1479ce9864050a21da05addb62c1858","verifier":"a-root","branch":"main","commit_sha":"0c1f80acacb6f131acfa7505228a7c0e653cc764"}
