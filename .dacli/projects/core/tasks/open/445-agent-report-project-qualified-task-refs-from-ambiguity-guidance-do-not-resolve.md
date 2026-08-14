---
id: t-01KZYQ5EMEJQ52K002KXT47S38
kind: task
created: 2026-08-13T23:26:22Z
created_by: a-root
owner: a-root
github:
  issue: 628
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# [agent-report] project-qualified task refs from ambiguity guidance do not resolve
## Context
Adopted from GitHub issue #628.

From the supabase task worktree, 'dacli task check 001 --n 1' reported ambiguity and suggested supabase/001-define-the-firebase-to-supabase-migration-contract-and-rollback-plan; using that exact ref returned not found. The globally unique bare slug resolves, then correctly enforces owner-only checking.

Implementation and claim boundary: `internal/store` owns `FindTask`, `TaskIndex`, shared reference parsing, ambiguity suggestions, and resolver regressions. `docs/FORMAT.md` owns the public task-reference grammar. Command slices already delegate to the shared store resolver; file a separate evidence-backed task if a command-specific exception is proven.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] The shared resolver accepts `<project>/<seq>`, `<project>/<padded-seq>`, `<project>/<NNN-slug>`, and `<project>/<slug>` for a task in that project.
- [ ] Project-qualified lookup never resolves a task from another project and reports unknown project/task distinctly enough to recover.
- [ ] Existing ULID, unique slug, unique sequence, and `NNN-slug` forms remain backward compatible.
- [ ] Every ambiguity suggestion emitted by task lookup round-trips through the same resolver and identifies exactly one task.
- [ ] `task check`, `accept`, and orchestration paths share the resolver behavior rather than implementing local exceptions.
- [ ] Regression tests cover two projects with identical sequences and slugs, valid qualified lookup, invalid qualifier, and suggestion round-trip.
- [ ] User documentation defines task-reference syntax and recommends ULIDs for generated mutating commands.
## Log
- 2026-08-14T00:36:35Z claimed by a-maintainer-1w0gkw
