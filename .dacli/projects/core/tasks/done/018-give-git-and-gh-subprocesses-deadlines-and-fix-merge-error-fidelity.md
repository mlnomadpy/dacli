---
id: t-01KY4R55VAF2QD7SG146G38AZM
kind: task
created: 2026-07-22T11:07:44Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 3, pessimistic: 5}
---
# Give git and gh subprocesses deadlines and fix Merge error fidelity

## Context
Real audit findings. Anchors:
- Every git/gh call is `exec.Command(...)+CombinedOutput()/Output()` with NO context/deadline: `gitx.go:14` (git Run, incl. push), `features/vcs/vcs.go:48`, `features/vcs/lifecycle.go:143` (gh pr create), `features/ghmirror/ghmirror.go:38` (gh issue/repo/auth). A network/auth-bound child can hang forever; under `dacli mcp serve` that blocks the whole stdio loop. Switch to `exec.CommandContext` with a sensible timeout (a few seconds for local git, longer for network gh). Keep the existing return/err shapes.
- `gitx.Merge` (gitx.go:111-122): on merge failure it runs `diff --diff-filter=U`, and if NO files are conflicted it substitutes `conflicts=['(merge failed; see git output)']` and returns `(conflicts, nil)`, discarding the real err. A non-conflict failure (missing branch, unrelated histories, index lock) is then misreported as a conflict and wrongly blocks the task (lifecycle.go:196). When --diff-filter=U yields no conflicted files, propagate the REAL error instead.

## Scope (STRICT) — touch ONLY:
- `internal/gitx/**`
- `internal/features/vcs/**`
- `internal/features/ghmirror/**`

## Staging discipline (IMPORTANT)
Do NOT `git add -A`. `git add` ONLY files under the scope dirs above plus this task's own file under `.dacli/projects/core/tasks/`. Commit via `dacli commit`. `go build ./...` + `go test ./internal/...` green before committing; paste the summary as `dacli note add finding`; then `dacli task check`.

## Acceptance
- [x] every git/gh exec uses context.CommandContext with a timeout so a hung child cannot block dacli mcp serve (gitx.go, features/vcs, features/ghmirror)
- [x] gitx.Merge propagates the real error when no files are conflicted, instead of misreporting a non-conflict failure as a conflict
- [x] committed on branch by an agent; go build + go test green
## Log
- 2026-07-22T11:46:22Z completed by a-root
