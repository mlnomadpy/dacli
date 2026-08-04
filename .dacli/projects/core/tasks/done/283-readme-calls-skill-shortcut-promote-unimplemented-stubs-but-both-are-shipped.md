---
id: t-01KZ6SM3KC6KTSQA7667MG88G6
kind: task
created: 2026-08-04T16:27:33Z
created_by: a-go-auditor-ag9yhz
owner: a-root
priority: should
estimate: "{optimistic: 0.2, probable: 0.5, pessimistic: 1.5}"
---
# README calls skill/shortcut promote unimplemented stubs, but both are shipped and tested
## Acceptance
- [x] README.md:85 no longer states skill promote / shortcut promote are 'honest stubs that refuse' — rewrite to reflect that both are implemented (or delete the sentence)
- [x] README.md:278 'Still stubbed (each refuses with an explanation): dacli skill promote, dacli shortcut promote' is removed or corrected; the surrounding command reference matches shipped behavior
- [x] verified against code: skillforge.go:86-138 and shortcuts.go cmdPromote implement the commands; clikit.Planned() has zero callers — the README no longer implies any command is a planned() stub
- [x] no remaining doc claims 'stub'/'planned' for a command that is actually implemented (cross-checked against docs/README.md:30, docs/SKILLS.md:3, docs/SHORTCUTS.md:3 which are already correct)
## Log
- 2026-08-04T20:08:49Z adopted by a-root (owner a-go-auditor-ag9yhz orphaned)
- 2026-08-04T20:09:05Z claimed by a-maintainer-d26jvm
- 2026-08-04T20:29:11Z accepted by a-root
- 2026-08-04T20:29:11Z verified by `grep -q 'skill promote' README.md` (exit 0)
- 2026-08-04T20:29:11Z completed by a-root
