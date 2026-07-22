---
id: t-01KY5A69Q70CN8METMEHCCPHGY
kind: task
created: 2026-07-22T16:22:55Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# G3: enrich dacli pr and mirror verify verdicts as PR review comments
## Context
Two pieces so human PR review sees what dacli already knows about a task.

1. **Enrich `dacli pr`.** `internal/features/vcs/lifecycle.go` `cmdPR` (:129) builds a plain body and calls `gh pr create` (:149). Extend the body:
   - Acceptance: `taskAcceptance(t)` (:167) already yields it — keep it.
   - Findings: append the task's finding notes (`store.ListNotes(w, t.Project, model.NoteFinding)` filtered to those whose `about` is this task) so the reviewer sees what the agents flagged.
   - `Fixes #<issue>`: read the task's mirrored issue number from its OWN frontmatter `github` block (`t.Doc.Front.GetBlock("github")` → `issue: N` — the same block ghmirror writes at push; do NOT import the ghmirror slice) and add a `Fixes #N` line so merging the PR closes the issue. Skip cleanly if the task isn't linked.

2. **Verify verdicts → PR review comments.** `internal/features/execution/verify.go` `cmdVerify` runs a panel and derives confirmed/refuted verdicts (:138), today written only to run outcomes. Make each panel run also record its verdict as a queryable note/event ABOUT the task (a finding or a small verdict record), THEN add a path (e.g. `dacli pr --with-verdicts`, or a `dacli verify ... --pr <n>` option) that posts the task's verdicts as PR review comments via `gh pr review`/`gh pr comment`. Use `exec.CommandContext` with a timeout for every gh call (per the selfreport/018 lesson — a hung gh must not block). Posting to GitHub is operator-triggered (a flag), never automatic.

Do NOT require a live gh call in tests — unit-test the body-assembly (acceptance + findings + Fixes line) and the verdict-record logic on fixtures.

## Scope (STRICT) — touch ONLY:
- `internal/features/vcs/lifecycle.go`
- `internal/features/execution/verify.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the two files above (+ a test file in one of those packages) plus this task's file. `go build ./...` + `go test ./internal/...` green. `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [x] dacli pr builds the PR body from the task's acceptance + linked findings and adds Fixes #<issue>; the verify panel's confirmed/refuted verdicts post as PR review comments so human review sees the model's adversarial checks
- [x] committed by an agent; build + test green
## Log
- 2026-07-22T17:20:49Z claimed by a-hwe0pzgt19
- 2026-07-22T17:28:01Z accepted by a-root
- 2026-07-22T17:28:01Z completed by a-root
