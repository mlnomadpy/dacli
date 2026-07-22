---
id: f-doc3-complete-on-branch-dacli-060-docs-github-md-documents-the-g-series
kind: note
note_kind: finding
created: 2026-07-22T18:12:13Z
created_by: a-36j29f5fcw
about: [[060]]
severity: moderate
---
# DOC3 complete on branch dacli/060: docs/GITHUB.md documents the G-series bidirectional mirror
Commit ff16373 by a-36j29f5fcw, staged ONLY docs/GITHUB.md (git add + dacli commit --no-add). Docs-only. All 3 acceptance criteria met, verified against internal/features/ghmirror/ghmirror.go and internal/features/vcs/lifecycle.go: (1) github push documented — tasks->issues (status:<folder> label via applyStatusLabel, close-on-done, github: backlink block), decisions->labeled 'decision' issues via mirrorDecisions (Chose/Rejected/Because body), findings->issue comments via mirrorFindings (per-finding <!-- dacli-finding: --> marker idempotency); github pull adopts human-authored (non-dacli-marker, unmapped) issues as tasks via store.CreateTask, idempotent by issue-number mapping; github sync = pull then push (cmdSync); dacli pr enrichment = prBody(acceptance + findings + Fixes #issue from github: block) and --with-verdicts posts verify-panel verdicts (verify-verdict: comment events, VerdictRecord/VerdictMarker) as a gh pr review --comment via postVerdicts. (2) Disclosure gate documented: github link --allow-public records github_public_confirmed on the project file; disclosureGate re-checks LIVE visibility at every push; pull is read-only/ungated; every push operator-triggered, no ship/commit/spawn path auto-publishes — a reader learns nothing publishes automatically. (3) committed by an agent, docs-only (56 insertions/18 deletions to docs/GITHUB.md). NOTE: hit the worktree-shadow trap (edits first landed in the MAIN checkout via absolute path); restored main via git checkout and re-authored in the worktree so the branch carries the change. Owner: verify and close via dacli task check 060 --n 1..3 + dacli task done 060, then dacli merge --task 060.
