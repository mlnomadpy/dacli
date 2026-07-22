---
id: t-01KY5JGGF42N46W4DBJPCBB1GS
kind: task
created: 2026-07-22T18:48:19Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# H2: project the role/skill roster to a browsable catalog (GitHub wiki page or docs), generated one-way
## Context
H1 (just shipped) gave roles/skills a `version:` and a git-history changelog (`store.FormatChangelog`, `store.LoadRoles`, `role show`/`skill show`). H2 projects that roster into ONE browsable catalog for humans — source of truth stays in `.dacli/`, the catalog is a generated read view.

Add a `dacli catalog [--out docs/ROSTER.md] [--publish-wiki]` command (put it in a small new slice `internal/features/catalog/` registered in `internal/cli/cli.go`, or in teamops — do NOT import another feature slice; read via `store.LoadRoles`, the skills store, and the changelog helpers):
- Generate a single markdown catalog: every ROLE (name, version, grant, kind, model, skills, one-line purpose, last-changed from the changelog) and every SKILL (name, version, purpose, est. tokens). Group and table it so it's scannable on GitHub.
- Default: write it to `docs/ROSTER.md` (versioned with the repo — the reliable catalog). Print a summary.
- `--publish-wiki` (optional, gated): also publish to the repo wiki — the wiki is a git repo (`<owner/repo>.wiki.git`); clone/pull it, write the page, push. Honor the SAME disclosure gate as `github push` (linked repo + PUBLIC consent). Best-effort; a wiki failure must not fail the docs write. Do NOT require a live network call in tests.
- One-way only: the catalog is generated FROM `.dacli/`; edits to a role/skill happen by editing the `.dacli` file (via PR), never on the catalog.

## Scope (STRICT) — touch ONLY:
- `internal/features/catalog/` (new) OR `internal/features/teamops/`
- `internal/cli/cli.go` (register, if a new slice)
- `docs/ROSTER.md` (generated output is fine to commit as a sample)

## Staging discipline
Do NOT `git add -A`. `git add` ONLY your new/edited files plus this task's file. `go build ./...` + `go test ./internal/...` green (arch_test forbids feature→feature imports). `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [x] dacli generates a human-browsable catalog of every role and skill (name, purpose, grant/kind, skills, current version, last-changed) and can publish it to the repo wiki or a docs page — one-way from the .dacli source; edits to the actual role/skill happen via PR, not on the catalog
- [x] gated/operator-triggered like the other GitHub projections; honors the disclosure gate
- [x] committed by an agent; build + test green
## Log
- 2026-07-22T20:48:16Z claimed by a-0bsqxp2kpx
- 2026-07-22T20:54:31Z accepted by a-root
- 2026-07-22T20:54:31Z completed by a-root
