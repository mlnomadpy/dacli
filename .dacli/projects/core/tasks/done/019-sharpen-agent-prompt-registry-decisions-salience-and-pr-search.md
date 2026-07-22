---
id: t-01KY4R55VH22TVFY69JV0MXE4F
kind: task
created: 2026-07-22T11:07:44Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 1, probable: 2, pessimistic: 3}
---
# Sharpen agent prompt registry: decisions, salience, and PR search

## Context
Real prompt-auditor findings. Anchors:
- `internal/prompts/tpl/protocol_preamble.md` documents `note add finding` and `ask` but NEVER `note add decision`, though decisions with rejected alternatives are a load-bearing artifact (mcp_tools.md:35, docs/PROMPTS.md). Add one bullet parallel to the finding bullet: when you choose an approach over a real alternative, `dacli note add decision … --rejected … --because …`.
- `internal/prompts/tpl/brief_header.md:2` wraps the data-not-instructions warning in `<!-- -->`, and `supervise_correction.md` wraps the unmet-criteria list the same way. HTML-comment syntax is exactly what models are trained to treat as inert/non-instructional — backwards for the highest-stakes anti-injection line and a must-fix correction list. Render both as an emphasized `**SYSTEM:** …` block instead. (The est-tokens comment line elsewhere in brief_header can stay a comment — salience is the concern only for these two.)
- `review_workflow.md:4` templates `gh pr list --search <Search>`; execution.go:611 binds `Search` to `t.ID` (task ULID), but `dacli pr` (features/vcs/lifecycle.go:143) writes only Seq+Title/Seq+Slug into the PR — the ULID matches nothing. Key the template off the branch/seq, and change the binding in `execution.go` to pass `BranchFor(t)` or `t.Seq` instead of `t.ID`.

## Scope (STRICT) — touch ONLY:
- `internal/prompts/tpl/**`
- the single `Search:` binding in `internal/features/execution/execution.go` (review_workflow render) — nothing else in that file.

## Staging discipline (IMPORTANT)
Do NOT `git add -A`. `git add` ONLY the prompt templates you changed, the one execution.go binding, plus this task's own file under `.dacli/projects/core/tasks/`. Commit via `dacli commit`. `go build ./...` + `go test ./internal/...` green before committing (prompt tests live in internal/prompts and internal/cli); paste the summary as `dacli note add finding`; then `dacli task check`.

## Acceptance
- [x] protocol_preamble.md tells agents to file decision notes (note add decision --rejected --because) parallel to the finding bullet
- [x] brief_header.md and supervise_correction.md render their security/must-fix lines as an emphasized SYSTEM block, not HTML comments an LLM may treat as inert
- [x] review_workflow.md no-PR-number search keys off the branch/seq that dacli pr actually writes (BranchFor/t.Seq), not t.ID; execution.go binds Search accordingly
- [x] committed on branch by an agent; go build + go test green
## Log
- 2026-07-22T11:46:22Z completed by a-root
