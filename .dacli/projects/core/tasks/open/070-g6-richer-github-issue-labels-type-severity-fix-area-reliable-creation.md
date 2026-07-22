---
id: t-01KY5JKS6V9AGZ0F8RDYF394E3
kind: task
created: 2026-07-22T18:50:06Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# G6: richer GitHub issue labels — type, severity fix, area, reliable creation
## Context
The public repo's finding-issues show `severity:unspecified` even though the finding notes DO carry `severity:` (e.g. `severity: moderate`) — so this is a read/map bug, not missing data. And label creation (`ensureLabel`) is flaky, so an issue-create with a not-yet-created label fails the push. Fix both and enrich the taxonomy. All in `internal/features/ghmirror/ghmirror.go`.

Anchors:
- `severityLabel` (:772) + the finding-issue push (:812–:846) read `dn.doc.Front.Get("severity")` and call `ensureLabel(w, sevLabel)`. DEBUG why it yields `unspecified` for a note that has `severity: moderate` — likely the wrong doc/field is read (confirm against a real note in `.dacli/projects/core/notes/findings/`). Map correctly to `severity:major|moderate|minor`.
- `ensureLabel` (:587) is best-effort and can miss under a flaky network. PRE-CREATE the full label set (with stable colors) ONCE at the start of a push — `finding`, `decision`, `type:finding|type:task|type:decision`, `severity:major|moderate|minor`, and the `area:*` labels you emit — so no issue-create ever fails on a missing label.
- Add `type:` labels: `type:finding` on finding-issues, `type:task` on task-issues (the main mirror loop), `type:decision` on decision-issues (:709–:729).
- Add a best-effort `area:<slice>` label: parse the first `internal/<...>` path out of the finding body (e.g. `internal/features/ghmirror` → `area:ghmirror`, `internal/store` → `area:store`); for task-issues, derive from the task's project. Skip cleanly when no area is detectable.
- Keep all gh calls through `gh(w,...)` (context timeout).

## Scope (STRICT) — touch ONLY: `internal/features/ghmirror/ghmirror.go` (+ test in the same package)
## Staging discipline
Do NOT `git add -A`. `git add` ONLY ghmirror.go (+test) plus this task's file. `go build ./...` + `go test ./internal/...` green; unit-test the severity mapping + area-from-path parsing on fixtures, NO live gh. `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [ ] severity labels are correct: a finding note's severity: (major/moderate/minor) maps to severity:<sev>, never the broken 'unspecified' seen on the public repo
- [ ] issues carry a type: label (type:finding | type:task | type:decision) and a best-effort area:<slice> label derived from the finding's file path or the task's project
- [ ] the full label set (colors) is pre-created reliably before any issue-create uses it, so a missing label never fails a push (the ensureLabel flakiness)
- [ ] committed by an agent; go build + go test ./internal/... green; unit-tested label mapping on fixtures, no live gh
## Log
- 2026-07-22T18:50:47Z claimed by a-rnrt9qqkyx
