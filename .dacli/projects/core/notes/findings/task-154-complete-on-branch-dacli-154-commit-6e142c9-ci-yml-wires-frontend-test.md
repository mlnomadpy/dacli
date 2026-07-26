---
id: f-task-154-complete-on-branch-dacli-154-commit-6e142c9-ci-yml-wires-frontend-test
kind: note
note_kind: finding
created: 2026-07-26T20:58:30Z
created_by: a-azfs9hw109
about: [[154]]
severity: major
---
# task 154 complete on branch dacli/154-..., commit 6e142c9: ci.yml wires frontend test+lint, all 4 acceptance criteria verified locally
Committed 6e142c9 (PR #262, https://github.com/mlnomadpy/dacli/pull/262, auto-merge queued). Change: .github/workflows/ci.yml 'test' job gains two steps after 'build frontend' and before 'gofmt' — 'test frontend' (working-directory internal/features/dashboard/ui, run: npm run test:unit) and 'lint frontend' (same dir, run: npm run lint). (1) test:unit wired right after the frontend build step, verified by actually running it in-worktree via 'dacli run --cmd' with an absolute path (see decision note) — 73/73 tests pass across 15 files. A failing test propagates exit status 1 to the shell step, which GitHub Actions treats as a failed step -> failed job, same mechanism as the existing go test step. (2) lint wired as its own step running 'eslint . --max-warnings 0' (package.json:15) — ran clean, 0 warnings/errors. (3) verified red-then-green by editing BurnRate.test.ts:14 to assert an impossible string, re-running npm run test:unit in-worktree -> 1/73 tests failed, dacli run exited status 1 (the same signal a CI step failure produces); reverted, git diff on the file is empty, re-ran -> 73/73 green again. (4) the new 'test frontend'/'lint frontend' steps are distinct steps placed BEFORE gofmt/vet/test(go)/build in the same job's linear step list; GitHub Actions job steps run sequentially and any step's non-zero exit fails the whole job immediately, so a frontend regression fails the job without needing or duplicating the existing 'go test' step. cross-compile job untouched (no test/lint need there). go build ./..., go vet ./..., gofmt -l . all clean on this branch.
