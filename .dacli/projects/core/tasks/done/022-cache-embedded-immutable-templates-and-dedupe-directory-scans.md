---
id: t-01KY4VTV2PEAT3SPZX1RBSAXED
kind: task
created: 2026-07-22T12:12:00Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 3, pessimistic: 5}
---
# Cache embedded immutable templates and dedupe directory scans
## Context
Real audit finding. Embedded templates are immutable at runtime but re-read/re-parsed every call. Anchors:
- `internal/prompts/prompts.go:84` MCPDesc re-reads embedded `tpl/mcp_tools.md` and full `mdstore.Parse()`s it on EVERY call — when the MCP server registers N tools it parses the whole doc N times. Hoist behind `sync.Once` into a parsed map.
- `internal/prompts/prompts.go:36` Render re-parses the template text each call; a content-keyed compiled-template cache removes it.
- `internal/gates/gates.go` Advance (~:263) calls `store.LoadProject` twice — Status/Stage already loaded it (~:230). Reuse the load.
- `internal/skills/skills.go` load() (~:103) + mainFile (~:62) each `os.ReadDir` the same dir — mainFile discards the entry list, load re-reads it. Have mainFile return the already-read entries so the dir is scanned once.

## Scope (STRICT) — touch ONLY:
- `internal/prompts/prompts.go`  (NOT tpl/*)
- `internal/gates/gates.go`
- `internal/skills/skills.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the three files above plus this task's file under `.dacli/projects/core/tasks/`. `go build ./...` + `go test ./internal/...` green before committing (the cli TestMain clears DACLI_AGENT so the suite passes for you). Paste the summary as `dacli note add finding`, `dacli commit` (author attribution), then `dacli task check`.

## Acceptance
- [x] prompts.MCPDesc parses the embedded mcp_tools.md ONCE (sync.Once to a map), not on every tool registration
- [x] prompts.Render avoids re-parsing identical template text on repeat calls (content-keyed compiled-template cache)
- [x] gates.Advance does not call store.LoadProject twice (Status already loaded it at :230)
- [x] skills load() reads each skill dir ONCE: mainFile returns the entries it read instead of load re-reading the same dir
- [x] committed on branch by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T12:22:25Z completed by a-root
