---
id: f-dacli-integrate-pr-pr-auto-cannot-advance-an-already-open-pr-gh-pr-create-runs
kind: note
note_kind: finding
created: 2026-08-04T11:05:49Z
created_by: a-integrator-ydcqew
about: "[[250]]"
severity: major
---
# dacli integrate --pr / pr --auto cannot advance an already-open PR: gh pr create runs first and hard-fails on 'already exists'
The integrator's sanctioned merge path is blocked for every PR the loop already opened. prIntegrateTask (internal/features/vcs/lifecycle.go:1103) unconditionally calls openPR (gh pr create) BEFORE reaching the check-gated merge (step 3c, lifecycle.go:1163) or the --auto queue (step 3a, lifecycle.go:1121). cmdPR does the same: openPR at lifecycle.go:221 runs before the --auto queue at lifecycle.go:234. When a PR already exists, gh pr create returns a non-network error and both commands abort with 'integrated 0 branch(es)'. Observed: 'dacli integrate --tasks 248 --into main --pr --auto' -> 'gh pr create failed: a pull request for branch dacli/248-... into main already exists: https://github.com/mlnomadpy/dacli/pull/287'. Same for 'dacli pr --task 248 --auto'. Net effect: the integrator cannot enable auto-merge on, or check-gate-merge, any of the 6 open PRs the loop left awaiting merge. gh is not directly runnable in this sandbox, so there is no fallback. Fix: detect an existing open PR (gh pr view) and skip create, proceeding to the merge/auto-merge gate.
