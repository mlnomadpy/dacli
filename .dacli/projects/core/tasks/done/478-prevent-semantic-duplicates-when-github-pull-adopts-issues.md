---
id: t-01M0CZAN6QKQC26961BMZMF79N
kind: task
created: 2026-08-19T12:18:23Z
created_by: a-root
owner: a-root
github:
  issue: 718
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Prevent semantic duplicates when GitHub pull adopts issues
## Context
Adopted from GitHub issue #718.

## Problem

`dacli github pull` skips issues that already have a mapping or dacli marker, but it does not apply the semantic near-duplicate checks used by local task creation. A human-authored GitHub issue can therefore be adopted beside an already-existing local task describing the same work, fragmenting ownership, estimates, dependencies, and closure.

## Design direction

Treat inbound adoption as a reconciliation decision. Before creating a task, compare the issue against existing project tasks using the same deterministic duplicate policy as task creation. Exact/high-confidence matches should link or refuse with an actionable preview; uncertain matches should remain explicit findings for operator resolution rather than silently merging records.

## Acceptance criteria

- `dacli github pull --dry-run <project>` reports create, already-mapped, exact-match/link, possible-duplicate, and refused outcomes without mutating state.
- The real pull does not create a second task for an exact deterministic match and preserves one GitHub mapping.
- Possible semantic duplicates fail closed or require an explicit audited resolution; they are never silently merged.
- Duplicate detection is shared with task creation rather than maintained as a divergent second heuristic.
- Closed/done/history tasks and cross-project candidates have documented, tested behavior.
- Repeated pull is idempotent after linking or resolution.
- Tests cover exact-title normalization, marker/mapping matches, near-duplicate ambiguity, cross-project isolation, and unchanged unrelated issues.
- Mutation proof demonstrates disabling the duplicate gate makes the focused test fail.

## Acceptance
- [x] `dacli github pull --dry-run <project>` reports create, already-mapped, exact-match/link, possible-duplicate, and refused outcomes without mutating state.
- [x] The real pull does not create a second task for an exact deterministic match and preserves one GitHub mapping.
- [x] Possible semantic duplicates fail closed or require an explicit audited resolution; they are never silently merged.
- [x] Duplicate detection is shared with task creation rather than maintained as a divergent second heuristic.
- [x] Closed/done/history tasks and cross-project candidates have documented, tested behavior.
- [x] Repeated pull is idempotent after linking or resolution.
- [x] Tests cover exact-title normalization, marker/mapping matches, near-duplicate ambiguity, cross-project isolation, and unchanged unrelated issues.
- [x] Mutation proof demonstrates disabling the duplicate gate makes the focused test fail.
## Log
- 2026-08-19T13:24:13Z claimed by a-maintainer-ycbqam
- 2026-08-19T14:14:15Z accepted by a-root
- 2026-08-19T14:14:15Z verified by `GOCACHE=/tmp/dacli-gocache-task478 go test -race ./internal/features/ghmirror` (exit 0) in branch main at 73f64cd — proves that tree builds, not that the work is in trunk
- 2026-08-19T14:14:15Z deliverable: dacli/478-prevent-semantic-duplicates-when-github-pull-adopts-issues is merged into main
- 2026-08-19T14:14:15Z completed by a-root
- 2026-08-19T14:23:32Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/738 (event 01M0D4RWAFJFBHZ8NNWKSZ23S4)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-gocache-task478 go test -race ./internal/features/ghmirror","exit_code":0,"duration_ms":212,"artifact_hash":"sha256:e277ec2a808690c7a3146ad7dbce8d75df2126fbb0dc9bbba3642de554178717","verifier":"a-root","branch":"main","commit_sha":"73f64cdc6690e12cfdd27c0e8112f45491cf9e9b"}
{"command":"GOCACHE=/tmp/dacli-gocache-task478 go test -race ./internal/features/ghmirror","exit_code":0,"duration_ms":217,"artifact_hash":"sha256:e277ec2a808690c7a3146ad7dbce8d75df2126fbb0dc9bbba3642de554178717","verifier":"a-root","branch":"main","commit_sha":"73f64cdc6690e12cfdd27c0e8112f45491cf9e9b"}
