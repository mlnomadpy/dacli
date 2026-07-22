---
id: t-01KY4R55VQDVNDXCNBJERR017P
kind: task
created: 2026-07-22T11:07:44Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 1, probable: 2, pessimistic: 4}
---
# Remove spm and brief-assembly algorithmic waste

## Context
Real audit findings. Anchors:
- `internal/spm/criticalpath.go:217` — the `sort.Slice(ready, …)` sits INSIDE the `for len(ready)>0` loop, re-sorting the frontier every pop: O(V² logV) instead of O(V logV). Use a `container/heap` keyed on pos, or an order-preserving insert.
- `internal/spm/ambiguity.go:221` — `string(b[i+3:])` copies the whole remaining buffer per fenced block; use `bytes.Index` on the byte slice (zero-alloc search).
- `internal/brief/brief.go:332` — trim loop is `for EstimateTokens(b.render()) > budget { drop one }`; `render()` (brief.go:349) rebuilds the entire document + re-tokenizes every pass, so dropping k sections re-renders k+1 times on the brief-assembly hot path. Keep a running token total and subtract the dropped section's estimate instead of re-rendering from scratch. Preserve identical output (same sections dropped, same order).

## Scope (STRICT) — touch ONLY:
- `internal/spm/**`
- `internal/brief/brief.go`

## Staging discipline (IMPORTANT)
Do NOT `git add -A`. `git add` ONLY files under the scope above plus this task's own file under `.dacli/projects/core/tasks/`. Commit via `dacli commit`. `go build ./...` + `go test ./internal/...` green before committing (brief/spm have tests — behaviour must not change); paste the summary as `dacli note add finding`; then `dacli task check`.

## Acceptance
- [x] spm kahn no longer re-sorts the ready frontier every pop (heap or order-preserving insert): criticalpath.go
- [x] spm maskCode uses bytes.Index instead of copying the whole tail per fence: ambiguity.go
- [x] brief.trim subtracts the dropped section token estimate from a running total instead of re-rendering the whole brief each pass: brief.go
- [x] committed on branch by an agent; go build + go test green
## Log
- 2026-07-22T11:46:22Z completed by a-root
