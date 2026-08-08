---
id: f-dacli-init-silently-ignores-template-and-roster-two-spec-documented-brief
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-9y38s7w8e2
about: [[t-01KY59FNFK27A1084PQ8R2CJ5S]]
source_event: 01KY59QDNF14BCXYP41Y70XNJ4
---
# dacli init silently ignores --template and --roster: two spec-documented, Brief-advertised flags do nothing
wscore.go:13 Brief: 'init — Create a .dacli workspace (--template to seed a process)'. But cmdInit (wscore.go:17-29) reads ONLY --name; it never touches --template or --roster, and clikit.ParseFlags silently accepts unknown flags (no error, exit 0). Specs promise both: docs/TEMPLATES.md:17 & :185 ('dacli init --template <name> seeds a workspace with a process, not just empty folders'), docs/WALKTHROUGH.md:12 shows the FIRST-RUN command 'dacli init --name billing --template solo', and docs/TEAM.md:68 documents 'dacli init --roster software|research|solo'. No --roster handler exists anywhere (grep: only teamops 'team' Brief + team.Team roster type). Net: a new user following the walkthrough gets an empty workspace with no process/roster seeded and NO error. The workspace-template subsystem (stagegate/templates) attaches only via 'project add --template' (a different command); init's advertised seeding is unimplemented. Either implement init's --template/--roster seeding or drop the flag from the Brief and the three spec docs.
