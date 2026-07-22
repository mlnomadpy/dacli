---
id: f-e2-complete-on-branch-dacli-032-commit-6a2465d-claim-scoped-commits-enforced
kind: note
note_kind: finding
created: 2026-07-22T14:50:39Z
created_by: a-gnnd772rq8
about: [[032]]
severity: moderate
---
# E2 complete on branch dacli/032 — commit 6a2465d, claim-scoped commits enforced, all acceptance met
Committed 6a2465d by a-gnnd772rq8 (maintainer) on branch dacli/032-e2-claim-scoped-commits, staging ONLY internal/features/vcs/vcs.go (my recorded claim) via git add + dacli commit --no-add. ACCEPTANCE all satisfied and verified end-to-end with the rebuilt binary against the shared .dacli: (1) dacli commit now reads the committing agent's --claim from the spawn's run record — agentClaims(w,id.ID) scans w.RunsDir() newest-first for the proc.txt whose child==id.ID with non-empty procmon.Record.Claims (vcs.go:184) — and REFUSES (clikit.Refusedf, exit 3) any staged CODE file outside that claim, naming the files; inClaimScope (vcs.go:214) always allows paths under .dacli/ (task file + workspace crumbs) and uses procmon.PathsOverlap for code files. This kills the 'do NOT git add -A, stage ONLY these files' boilerplate. (2) the claim is read from where the spawn already recorded it (no new flag); --force overrides with a loud stderr note naming the out-of-scope files (verified: 'warning: --force committing 1 file(s) OUTSIDE your claim [...]: scratch...' then commit succeeded); an agent with NO recorded claim is warned once and proceeds (not hard-blocked), so pre-E2/manual commits still work. (3) committed on branch by an agent; go build ./... clean; go test ./internal/... all green (incl. internal/cli, TestFeatureSlicesAreIsolated — vcs importing procmon is allowed, procmon is a shared entity not a feature slice). Verified: staging an out-of-claim file -> exit 3 refusal listing it; --force -> loud note + commit; my own in-scope commit (vcs.go only) PASSED the very check it adds (dogfood). Owner: verify and close via dacli task check/done + dacli merge --task 032.
