---
id: t-01KZ70HYMAWWA5R0ZYKAGTWY6E
kind: task
created: 2026-08-04T18:28:42Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 0.5, probable: 1, pessimistic: 3}"
---
# github push is broken: repoView passes --repo to gh repo view, which does not accept it
## So that
the outbound mirror works at all, instead of failing on its first call
## Acceptance
- [ ] repo view takes the repository as a positional argument, the form gh actually supports
- [ ] a test exercises the full push prologue against a stub that rejects an unsupported flag, so a wrong argument shape fails in CI rather than on a live repo
- [ ] every other ghRepo call site is checked against whether that subcommand accepts --repo
## Log
