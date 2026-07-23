---
id: t-01KY5VX10Q350YDM6MP89Y2KWB
kind: task
created: 2026-07-22T21:32:26Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# I1: dacli ship --pr — push branches, open enriched PRs, merge via gh (PR-first integration)
## Context
Today `dacli ship` (internal/features/ship/ship.go) and `dacli integrate` (internal/features/vcs/lifecycle.go) merge task branches LOCALLY with `git merge`. The operator wants PR-first integration — each done task's branch becomes a reviewable, issue-linked PR. All the pieces exist; wire them.

Add a `--pr` mode to `dacli ship` (ship shells out to its own binary, so it can call the existing commands):
- For each done task with a branch: `dacli push --task <ref>` (push the branch), then `dacli pr --task <ref>` (already builds the PR body from acceptance + linked findings + `Fixes #<issue>` and posts verify verdicts as review comments — G3), then merge it: `gh pr merge <n> --squash --delete-branch` (or `--merge`). Report each PR URL.
- `--no-merge`: open the PRs and STOP (leave them for human review) instead of auto-merging.
- Fallback: if GitHub is unreachable (push or gh fails with a network error), warn and fall back to the local `git merge` path so a wave still lands offline — this is the documented fallback, not silent.
- Keep the default (no `--pr`) local-merge behaviour unchanged.

## Scope (STRICT) — touch ONLY: `internal/features/ship/ship.go` (+ test)
## Staging: do NOT `git add -A`; add only ship.go (+test) + this task file. `go build`/`go test ./internal/...` green; test the flag routing + fallback on fixtures, no live gh. `dacli commit`; box-check owner-only.

## Acceptance
- [x] dacli ship --pr (and integrate --pr) pushes each done task's branch, opens a PR via dacli pr (body from acceptance + findings + Fixes #issue; verify verdicts as review comments), and merges via gh pr merge — instead of a local git merge
- [x] gated + operator-triggered; falls back to local merge with a warning when GitHub is unreachable; --no-merge opens the PRs and stops for human review
- [x] committed by an agent; build + test green
## Log
- 2026-07-22T21:33:27Z claimed by a-c79p0msrw8
- 2026-07-22T21:45:12Z accepted by a-root
- 2026-07-22T21:45:12Z completed by a-root
- 2026-07-22T23:52:35Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/39 (event 01KY5WK812JWDTFXGNJS6D12Q0)
