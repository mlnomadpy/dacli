---
id: t-01KY5GY1JT9W3HTQG62KESK28N
kind: task
created: 2026-07-22T18:20:45Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# G5: mirror findings as standalone GitHub issues (severity label, idempotent, backlink)

## Context
Reviewers file findings locally; the operator wants each FINDING to become its own GitHub ISSUE (triageable, severity-labeled) — distinct from G4, which posts findings as COMMENTS on a task's issue. Build in `internal/features/ghmirror/ghmirror.go`, reusing existing patterns — do NOT reinvent the gate or idempotency.

- Add a `--findings-as-issues` flag to `dacli github push <project>` (cmdPush). When set, for every finding note in the project (`store.ListNotes(w, project, model.NoteFinding)`), open ONE GitHub issue: title from the finding's title, body = the finding detail + a backlink to the local note, labels `finding` + `severity:<major|moderate|minor>` (create labels best-effort). Idempotent: embed a per-finding marker (reuse the `marker`/`searchByMarker` machinery keyed on note id + workspace id), search before create, and write the issue number back onto the finding note's frontmatter. A re-push must NEVER duplicate.
- Honor the SAME disclosure gate cmdPush already enforces (linked repo + PUBLIC consent via `--allow-public` + live visibility re-check) — do not weaken it.
- Keep G4's finding-comment path intact for the default (no-flag) behavior; `--findings-as-issues` is the standalone-issue mode.
- All gh via `gh(w,...)` (context timeout). Unit-test marker/idempotency/label-mapping on fixtures; NO live gh in tests.

## Scope (STRICT) — touch ONLY: `internal/features/ghmirror/ghmirror.go` (+ test in same package)
## Staging: do NOT `git add -A`; add only ghmirror.go (+test) + this task file. `go build` + `go test ./internal/...` green. `dacli note add finding`, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [ ] dacli github push opens one GitHub ISSUE per finding note (title from the finding, body = detail + backlink, label finding + severity), marker-idempotent so a re-push never duplicates; the issue number is written back onto the finding note
- [ ] this is distinct from G4's finding-as-comment; a flag or the default selects issue-per-finding for audit-style findings; honors the same disclosure gate (linked repo + visibility consent), operator-triggered only
- [ ] committed by an agent; go build + go test ./internal/... green; unit-tested on fixtures, no live gh
## Log
- 2026-07-22T18:23:19Z claimed by a-n03m4hw62x
