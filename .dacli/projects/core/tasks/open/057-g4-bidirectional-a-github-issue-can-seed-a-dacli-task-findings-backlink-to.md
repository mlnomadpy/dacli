---
id: t-01KY5A69QKKVJ992NJX7W5VQEA
kind: task
created: 2026-07-22T16:22:55Z
created_by: a-root
owner: a-root
priority: could
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# G4: bidirectional — a GitHub issue can seed a dacli task; findings backlink to issues
## Context
This closes the GitHub loop: outbound (tasks/decisions/verdicts → GitHub) already works via `github push`; G4 adds INBOUND and finding backlinks. Both in `internal/features/ghmirror/ghmirror.go`.

1. **Inbound: `github pull <project>`** (currently a `clikit.Planned` stub at :44). Implement it: `gh issue list` on the linked repo, and for each issue that has NO dacli marker (i.e. human-authored, not one we mirrored — check the body for the `<!-- dacli:... -->` marker via the existing `marker`/`searchByMarker` machinery), create a local task with `store.CreateTask(w, actor, project, title, opts)` (see planning.go:121 for the call shape), seeding title/body from the issue and writing the `github: issue/repo` block back onto the new task so it is linked (no re-import on the next pull — idempotent). Skip issues that already map to a local task.
2. **Findings → issue comments.** In `cmdPush` (:141), after a task's issue is ensured, post the task's finding notes (`store.ListNotes(w, project, model.NoteFinding)` filtered to `about==task`) as comments on the mirrored issue via `gh issue comment <n> --body ...`, idempotently (embed a per-finding marker in the comment and skip if already present — reuse the marker pattern so a re-push doesn't duplicate comments). This makes a finding a human sees in GitHub.
3. Optionally point `github sync` (:43) at pull-then-push instead of leaving it a stub.

All gh calls through the `gh(w, ...)` helper (has a context timeout) and behind the SAME disclosure gate cmdPush uses. Operator-triggered only. Do NOT require a live gh call in tests — unit-test the marker/idempotency/skip logic on fixtures.

## Scope (STRICT) — touch ONLY:
- `internal/features/ghmirror/ghmirror.go` (+ test in the same package)

## Staging discipline
Do NOT `git add -A`. `git add` ONLY ghmirror.go (+ test) plus this task's file. `go build ./...` + `go test ./internal/...` green. `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [ ] dacli can import a GitHub issue as a task (adopt-style), and a finding filed against a task references the mirrored issue so a human sees it in GitHub
- [ ] committed by an agent; build + test green
## Log
- 2026-07-22T17:28:47Z claimed by a-sdpxn53045
