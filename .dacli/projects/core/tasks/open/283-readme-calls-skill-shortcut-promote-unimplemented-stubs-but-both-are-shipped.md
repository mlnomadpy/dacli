---
id: t-01KZ6SM3KC6KTSQA7667MG88G6
kind: task
created: 2026-08-04T16:27:33Z
created_by: a-go-auditor-ag9yhz
owner: a-go-auditor-ag9yhz
priority: should
---
# README calls skill/shortcut promote unimplemented stubs, but both are shipped and tested
## Acceptance
- [ ] README.md:85 no longer states skill promote / shortcut promote are 'honest stubs that refuse' — rewrite to reflect that both are implemented (or delete the sentence)
- [ ] README.md:278 'Still stubbed (each refuses with an explanation): dacli skill promote, dacli shortcut promote' is removed or corrected; the surrounding command reference matches shipped behavior
- [ ] verified against code: skillforge.go:86-138 and shortcuts.go cmdPromote implement the commands; clikit.Planned() has zero callers — the README no longer implies any command is a planned() stub
- [ ] no remaining doc claims 'stub'/'planned' for a command that is actually implemented (cross-checked against docs/README.md:30, docs/SKILLS.md:3, docs/SHORTCUTS.md:3 which are already correct)
## Log
