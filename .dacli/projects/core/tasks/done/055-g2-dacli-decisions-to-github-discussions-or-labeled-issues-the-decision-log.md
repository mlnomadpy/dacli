---
id: t-01KY5A69PWJ2DC6QW4CMFYZ62H
kind: task
created: 2026-07-22T16:22:55Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 4, pessimistic: 6}
---
# G2: dacli decisions to GitHub Discussions (or labeled issues) — the decision log humans can read
## Context
Reality check first: `dacli github push` (internal/features/ghmirror/ghmirror.go:138) ALREADY mirrors tasks→issues idempotently, backlinks the issue number onto the task, closes the issue on done, and is disclosure-gated (link --allow-public + live PRIVATE/PUBLIC re-check). This task EXTENDS that mirror; follow its exact patterns — do NOT reinvent them.

Two pieces, both in `internal/features/ghmirror/ghmirror.go`:

1. **G1 residual — status labels.** `cmdPush` closes done issues but sets no per-status label. Add a `status:<folder>` label (e.g. `status:doing`, `status:blocked`, `status:done`) to each mirrored issue reflecting the task's status folder, via `gh issue edit <n> --add-label ...` (create the label if missing with `gh label create`, best-effort). Keep it idempotent (don't stack duplicate labels).

2. **G2 — decisions → GitHub.** Add mirroring of DECISION notes. Prefer GitHub **Discussions** if reachable via `gh api graphql`; if that's awkward, fall back to **issues labeled `decision`** (the acceptance explicitly allows either). REUSE the existing idempotency machinery: a `marker(...)`-style hidden comment keyed on the note id + workspace id (marker at :222), `searchByMarker` (:253) before create, and write the created number back onto the note's frontmatter (mirror how tasks store `github: issue/repo` at :203). Body = the decision's choice + rejected alternative + because (the WHY — that is the whole point). Honor the SAME disclosure gate as cmdPush (repo linked + visibility consent; refuse on PUBLIC without recorded consent). This runs only on an explicit `dacli github push` (already operator-triggered) — do NOT auto-run on ship.

Use the `gh(w, ...)` helper (:47) for all gh calls (it already has a context timeout). Do not require a live gh call in tests — unit-test the marker/idempotency/label-dedup logic on fixtures.

## Scope (STRICT) — touch ONLY:
- `internal/features/ghmirror/ghmirror.go` (+ a test file in the same package)

## Staging discipline
Do NOT `git add -A`. `git add` ONLY ghmirror.go (+ test) plus this task's file. `go build ./...` + `go test ./internal/...` green. `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [x] each dacli decision note (choice + rejected alternative + because) materializes as a GitHub Discussion/issue so the WHY is visible and searchable to the team, backlinked to the dacli decision
- [x] committed by an agent; build + test green
## Log
- 2026-07-22T17:04:40Z claimed by a-wp88a6y71z
- 2026-07-22T17:11:05Z accepted by a-root
- 2026-07-22T17:11:05Z completed by a-root
- 2026-07-22T18:23:33Z status done proposed by a-wp88a6y71z, applied (event 01KY5CWNV1SRK1V4TD3GD61FDW)
