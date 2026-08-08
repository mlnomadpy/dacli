---
id: f-028-complete-on-branch-dacli-028-commit-506499c-all-4-acceptance-met
kind: note
note_kind: finding
created: 2026-07-22T13:42:01Z
created_by: a-q2w31150s0
about: [[028]]
severity: moderate
---
# 028 complete on branch dacli/028 — commit 506499c, all 4 acceptance met
Committed 506499c by a-q2w31150s0 (maintainer) via git add + dacli commit --no-add, staging ONLY the 2 scoped files: internal/features/execution/execution.go and internal/features/insight/insight.go (the .go changes live in the dacli/028 worktree; the shared .dacli task-file/notes stay in the main checkout, so they are not on this branch — the owner integrates). ACCEPTANCE, all satisfied: (1) spawn --advise prints a suggested budget/sizing from the calibrated store.Band{role,model,runtime} (n>=10 -> empirical median + p10-p90 projected to hours via Te; else 'no band history yet') AFTER role/model/runtime/task resolve but BEFORE agentid.Spawn — and the spawn proceeds unchanged (axiom 3). (2) taint status shown before launch via store.Taint('external:') + TaintResult.ExposedBriefs: 'task NNN is in the blast radius of <origins>' if the task slug is exposed, else 'taint: clean'. (3) next --parallel annotates shown tasks with scope-matched store.WorkspaceLessons -> 'lesson L applies — consider role R' (R = role whose scope glob covers a path the lesson cites, else a role named in the lesson); HINT only, never an assignment. (4) go build ./... clean; go test ./internal/... all green incl. TestFeatureSlicesAreIsolated. NOTE: percentile is a documented local copy in execution.go because the arch test forbids execution importing insight and the strict scope forbids a shared spm/store helper (see decision note). Smoke of the rebuilt binary was blocked by the headless sandbox (arbitrary-path exec needs approval), so verified via build+unit tests only. Owner: verify and close via dacli task check/done + dacli merge --task 028.
