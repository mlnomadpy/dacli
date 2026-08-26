---
id: t-01M0ZCAPQAVDTVV82DNRW7969Q
kind: task
created: 2026-08-26T15:51:56Z
created_by: a-root
owner: a-root
github:
  issue: 794
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# [agent-report] spawn silently retains only the last repeated claim flag
## Context
Adopted from GitHub issue #794.

Reproduced twice with detached isolated-worktree spawns. Commands supplied two explicit --claim flags, but the resulting run retained only the second path. Implementations correctly modified files under the first claim, then dacli commit refused with policy exit 3; owner recovery required worktree reclaim with exact paths. Expected: repeated --claim flags accumulate, or the CLI rejects repeated syntax and documents a single comma-separated form before spawning. Acceptance criteria: a spawn with --claim path-a --claim path-b records both claims; commit accepts files under either and rejects a third path; the resolved claims are printed in dry-run/advise output. Non-goal: weakening claim enforcement.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] `dacli spawn --claim path-a --claim path-b` records both normalized claims instead of retaining only the last flag.
- [ ] `dacli commit` accepts files under either recorded claim and refuses a third unclaimed path with policy exit 3.
- [ ] Spawn dry-run or advise output prints the complete resolved claim set.
- [ ] Repeated and comma-separated claim forms have documented, regression-tested semantics without weakening overlap enforcement.
- [ ] Mutation evidence and the full repository verification gates pass.
## Log
