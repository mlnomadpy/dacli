---
id: t-01KZRRWCD0WBP4V8X67YJ10PQP
kind: task
created: 2026-08-11T16:00:55Z
created_by: a-go-auditor-7sx6nh
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# LandingOfRef checks a stale origin/<trunk> and never the local ref, so a local-merge ship records every task it just merged as 'NOT in <trunk> - closed anyway'
## So that
the record is the product; on the DEFAULT ship --push path (local merge into main with an origin remote present) recordWaveLanding->LandingOfRef measures the captured branch sha against origin/main, which is stale at that moment because the merge is local-only and push happens later (ship.go:252 runs before step 4 at :283), so LandingOfRef short-circuits on the first-resolving ref (origin/main) at landing.go:83-95 and returns LandingUnlanded WITHOUT ever checking refs/heads/main, which does contain the merge -- committing a permanent, false 'NOT in main - closed anyway' line (landing.go:114) on every task ship itself just landed. This is the residual of dacli 329 on a path that fix (timing: record after integrate) does not cover: the bug here is ref-selection, not timing
## Acceptance
- [x] LandingOfRef (internal/store/landing.go:79-97) no longer returns LandingUnlanded for a commit that is an ancestor of the LOCAL trunk ref (refs/heads/<trunk>) merely because origin/<trunk> exists but is behind: when the resolved trunk refs disagree, a commit reachable from EITHER the local or the remote trunk ref is reported LandingLanded, not LandingUnlanded
- [x] A test drives the default (non --pr) ship local-merge path -- or LandingOfRef directly -- with an origin/<trunk> that is stale (behind refs/heads/<trunk>, which contains the just-made --no-ff merge) and asserts the trajectory line recordWaveLanding (ship.go:662-676) stamps reads that the deliverable is merged/landed, NOT 'exists but is NOT in <trunk> - closed anyway'
- [x] Red-green demonstrated in the commit message or a finding: the new test fails on current HEAD and passes after the fix
- [x] The genuinely-unlanded case (a branch merged nowhere) still reports LandingUnlanded, and the --pr path (origin/<trunk> carries the merge via git pull --ff-only) still reports LandingLanded -- no regression to the dacli 329 / 342 behavior
## Log
- 2026-08-11T17:17:17Z adopted by a-root (owner a-go-auditor-7sx6nh orphaned)
- 2026-08-11T17:17:17Z accepted by a-root
- 2026-08-11T17:17:17Z closed WITHOUT verification — no --verify command was given
- 2026-08-11T17:17:17Z deliverable: no dacli/362-landingofref-checks-a-stale-origin-trunk-and-never-the-local-ref-so-a-local branch — nothing to check against main
- 2026-08-11T17:17:17Z completed by a-root
