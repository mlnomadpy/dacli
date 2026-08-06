---
id: d-084-filed-task-159-loop-never-fetches-ff-local-main-after-async-auto-pr-merges
kind: note
note_kind: decision
created: 2026-07-26T21:56:01Z
created_by: a-s4764r5zf3
about: [[084]]
github:
  issue: 348
  repo: mlnomadpy/dacli
---
# 084: filed task 159 (loop never fetches+ff local main after async --auto PR merges) as the single highest-value evidence-based change
## Chose
084: filed task 159 (loop never fetches+ff local main after async --auto PR merges) as the single highest-value evidence-based change
## Rejected
Filing the reviewer.md-lacks-role_kind burn-Rate dilution finding, or re-filing an already-covered item
## Because
The reviewer.md gap (f-reviewer-md-role...-slip-past-the-burn) is real but MINOR and data-only: a one-line 'role_kind: reviewer' backfill into .dacli/roles/reviewer.md, workspace state not code, and its diluting sibling population (verify-panel seats) is already excluded by the explicit verify_panel_seat marker. Task 159 is a MODERATE correctness/robustness defect in shipped orchestration code that task 114 EXPLICITLY scoped out as 'worth its own task': I verified in source that internal/gitx/gitx.go has NO Pull/Fetch/Sync/FastForward helper (only Push at gitx.go:219), the --auto merge path (lifecycle.go:757-767) returns with no later sync, the sole pull --ff-only (lifecycle.go:815) fires only on the synchronous gated path, and ship --push (ship.go:168) / gitx.Push (gitx.go:219-221) is a bare 'git push -u origin' with no pull/rebase retry. Once 114's deferred-push fix lets auto-merges land under branch protection, local main falls behind origin and the next push fails non-fast-forward with the error only best-effort-logged. Fixing the loop's own git reliability outranks a minor workspace-data backfill.
