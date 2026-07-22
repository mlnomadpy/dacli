---
id: t-01KY4VTV2WZ4AGAAWP50QNA7F3
kind: task
created: 2026-07-22T12:12:00Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 3, pessimistic: 5}
---
# Halve collab/vcs/teamops event-tree IO; expose parsed applied on Event
## Context
Real audit finding. `eventlog.List` (eventlog.go:96) already WalkDirs + ReadFiles + parses every event's `applied` field, but callers re-derive it. Anchors:
- `internal/eventlog/eventlog.go:28` — add the parsed `applied` bool to the `Event` struct and set it in List's parse (the Pending query path at :141 already reads it).
- `internal/features/collab/collab.go:169-188` cmdThreads calls `eventlog.List` TWICE (EventHelp :169, EventAnswer :173 = two full event-tree walks) and, inside the questions loop, `os.ReadFile(q.Path)` a THIRD time per question just to `strings.Contains("applied: true")`. Use the new `Event.Applied` and a single kind-filtered pass.
- `internal/features/vcs/vcs.go` cmdContrib (~:184,213) does two full List scans (commits, findings); `internal/features/teamops/teamops.go:113` cmdAgentTree does a full `List(Query{})`. Combine kind-filtered passes where a single walk suffices.

## Scope (STRICT) — touch ONLY:
- `internal/eventlog/eventlog.go`
- `internal/features/collab/collab.go`
- `internal/features/vcs/vcs.go`
- `internal/features/teamops/teamops.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the four files above plus this task's file under `.dacli/projects/core/tasks/`. `go build ./...` + `go test ./internal/...` green before committing (cli TestMain clears DACLI_AGENT). Paste the summary as `dacli note add finding`, `dacli commit`, then `dacli task check`. Behaviour must not change — these are pure I/O reductions.

## Acceptance
- [x] eventlog.Event carries the parsed applied flag (List already reads it); nothing re-opens an event file to check it
- [x] collab.cmdThreads stops os.ReadFile-ing each question a 3rd time and stops walking the event tree twice (one kind-filtered scan)
- [x] vcs.cmdContrib and teamops.cmdAgentTree avoid redundant full List scans where a single pass suffices
- [x] committed on branch by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T12:22:25Z completed by a-root
