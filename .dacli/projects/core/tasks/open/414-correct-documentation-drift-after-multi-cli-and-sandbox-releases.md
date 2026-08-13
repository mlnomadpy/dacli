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
- [ ] README.md, docs/index.md, docs/RUNTIMES.md, docs/ROSTER.md, and docs/WALKTHROUGH.md agree with shipped runtime and sandbox behavior.
- [ ] Runtime preset count and names match the executable preset registry.
- [ ] Generated roster content is regenerated from workspace state rather than manually forged.
- [ ] Volatile self-hosting statistics are removed or generated from a durable source.
- [ ] Documentation support tests and go test ./... pass.
## Log
