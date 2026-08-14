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
- [x] The shared resolver accepts `<project>/<seq>`, `<project>/<padded-seq>`, `<project>/<NNN-slug>`, and `<project>/<slug>` for a task in that project.
- [x] Project-qualified lookup never resolves a task from another project and reports unknown project/task distinctly enough to recover.
- [x] Existing ULID, unique slug, unique sequence, and `NNN-slug` forms remain backward compatible.
- [x] Every ambiguity suggestion emitted by task lookup round-trips through the same resolver and identifies exactly one task.
- [x] `task check`, `accept`, and orchestration paths share the resolver behavior rather than implementing local exceptions.
- [x] Regression tests cover two projects with identical sequences and slugs, valid qualified lookup, invalid qualifier, and suggestion round-trip.
- [x] User documentation defines task-reference syntax and recommends ULIDs for generated mutating commands.
## Log
- 2026-08-14T00:36:35Z claimed by a-maintainer-1w0gkw
- 2026-08-14T00:49:31Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/660 (event 01KZYVJ659KS4R736G4V1ESB2Y)
- 2026-08-14T00:50:42Z accepted by a-root (applied 1 proposal(s))
- 2026-08-14T00:50:42Z verified by `GOCACHE=/tmp/dacli-accept-445 go test ./...` (exit 0) in branch main at e10f64c — proves that tree builds, not that the work is in trunk
- 2026-08-14T00:50:42Z deliverable: dacli/445-agent-report-project-qualified-task-refs-from-ambiguity-guidance-do-not-resolve is merged into main
- 2026-08-14T00:50:42Z completed by a-root
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-accept-445 go test ./...","exit_code":0,"duration_ms":63271,"artifact_hash":"sha256:69be2f23924f486bad63c0299bbcd24a64c6b91a32cd6b3276de99085baa99a3","verifier":"a-root","branch":"main","commit_sha":"e10f64c840e64aa4dabaa202d9965cacad368037"}
