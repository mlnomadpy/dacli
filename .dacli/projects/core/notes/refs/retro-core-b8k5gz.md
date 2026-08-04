---
id: r-retro-core-b8k5gz
kind: note
note_kind: ref
created: 2026-08-04T11:28:11Z
created_by: a-root
about: "[[core]]"
---
# Retro: core
## Went well
- Invariant and mutation-style tests kept catching the class of bug this codebase actually produces: a rule applied in N places and missed in the N+1th. 242 (three diverged firstLine copies) and 243 (four integer-flag parsers) were both found that way, not by reading the feature they lived in.
- Spawning maintainers into isolated worktrees landed six independent fixes in one wave with no collisions, and each one arrived test-first with its pre-fix failure quoted.
- The integrator earned its keep by refusing to merge and filing a major finding instead: gh pr create runs before the merge gate, so integrate --pr could never merge a PR the loop had already opened. That is the bug that made me merge by hand all session.

## Didn't go well
- I merged by hand, with raw gh, instead of spawning the integrator role that exists for it — and only did so after being asked twice why not.
- dacli task add handed out seq 250 three times and 251 twice, because the allocator scans the working tree and a branch is a tree it cannot see. Main now carries two tasks numbered 250.
- team assign recommended junior for all six tasks in the wave, and junior cannot write: grant rw, runtime cc, no Edit or Write in the allowlist. The cheapest route was a route that could not do the work.

## Improve next time
- Spawn the integrator at the START of a wave, not after the PRs pile up — merging is a role, not an operator chore.
- Fix task 251 before widening parallelism: every --worktree spawn files against its own checkout, so seq collisions get worse with every agent added.
- Cross-check a role's runtime allowlist against its grant before routing implementation work to it, until 250 makes that a spawn-time refusal.

