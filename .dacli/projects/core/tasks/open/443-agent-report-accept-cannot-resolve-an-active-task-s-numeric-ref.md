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
- [ ] Generated task briefs and completion instructions use the task ULID for mutating commands such as `accept` and `task check`, never a workspace-ambiguous sequence alone.
- [ ] An active task can be accepted from its isolated worktree using the generated ULID command.
- [ ] Bare numeric references remain accepted when unique and return an ambiguity error without mutation when multiple projects share the sequence.
- [ ] Ambiguity guidance names only reference forms that the resolver actually accepts; project-qualified guidance is covered with issue #628 / task 445.
- [ ] Regression tests create two projects with the same sequence and prove the old generated numeric command fails while the fixed generated command resolves the assigned task.
- [ ] Documentation distinguishes stable machine references (ULIDs) from human shorthand.
## Log
- 2026-08-14T01:22:52Z claimed by a-maintainer-addy71
