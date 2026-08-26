---
id: t-01M0ZCAQ33YAXPS79D8EJ676KP
kind: task
created: 2026-08-26T15:51:56Z
created_by: a-root
owner: a-root
github:
  issue: 790
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] pr defaults to main instead of the linked repository default branch
## Context
Adopted from GitHub issue #790.

When a linked repository uses master as its GitHub default branch, dacli pr --task <ref> omitted --base and invoked PR creation against main, producing GraphQL errors: no commits between main and the task branch / base ref must be a branch. Re-running the same command with --base master succeeded. Expected: resolve the linked repository default branch (or effective project landing base) before PR creation.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] `dacli pr --task <ref>` resolves the explicit CLI base first, then configured landing base, then the linked repository default branch.
- [ ] A linked repository whose default branch is `master` creates the PR against `master` when no base override is configured.
- [ ] An explicit or configured non-default landing base is preserved and reported in dry-run and real execution.
- [ ] Failure to resolve an authoritative base fails closed before invoking PR creation and names the recovery action.
- [ ] Public-command tests cover default-branch discovery, configured override precedence, and remote failure.
- [ ] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-26T23:22:00Z claimed by a-claude-fixer-6jmk82
