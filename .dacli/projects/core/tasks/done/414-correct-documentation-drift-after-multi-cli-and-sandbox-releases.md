---
id: t-01KZXN0MYE7A0Z47HHAKMF3SRB
kind: task
created: 2026-08-13T13:29:33Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 570
  repo: mlnomadpy/dacli
---
# Correct documentation drift after multi-CLI and sandbox releases
## Acceptance
- [x] README.md, docs/index.md, docs/RUNTIMES.md, docs/ROSTER.md, and docs/WALKTHROUGH.md agree with shipped runtime and sandbox behavior.
- [x] Runtime preset count and names match the executable preset registry.
- [x] Generated roster content is regenerated from workspace state rather than manually forged.
- [x] Volatile self-hosting statistics are removed or generated from a durable source.
- [x] Documentation support tests and go test ./... pass.
## Log
- 2026-08-13T13:30:40Z claimed by a-fixer-xdgjvd
- 2026-08-13T14:04:24Z accepted by a-root
- 2026-08-13T14:04:24Z verified by `GOCACHE=/private/tmp/dacli-docs-414-accept go test ./docs ./internal/features/catalog` (exit 0) in branch main at 73aecf2 — proves that tree builds, not that the work is in trunk
- 2026-08-13T14:04:24Z completed by a-root
