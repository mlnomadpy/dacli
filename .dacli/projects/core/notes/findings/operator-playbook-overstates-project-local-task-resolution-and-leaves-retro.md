---
id: f-operator-playbook-overstates-project-local-task-resolution-and-leaves-retro
kind: note
note_kind: finding
created: 2026-08-19T11:46:08Z
created_by: a-root
about: "[[475]]"
severity: major
---
# Operator playbook overstates project-local task resolution and leaves retro command incomplete
docs/OPERATOR_PLAYBOOK.md currently says task references resolve within the selected project, but skills/dacli/references/workspace-tasks-projects.md and store.FindTask state direct lookup searches the whole workspace and ambiguous short refs fail; revise to say project-scoped list/schedule views isolate tasks while direct refs are workspace-wide. The cycle also prints `dacli retro <ref>` without required --well/--bad/--improve arguments; either show a valid command or describe it without presenting an incomplete invocation. Preserve GitHub as the primary human collaboration surface while calling the local dacli record the execution/evidence ledger.
