---
id: f-independent-forward-test-found-final-executable-ambiguities-in-task-475
kind: note
note_kind: finding
created: 2026-08-19T12:24:30Z
created_by: a-root
about: "[[475]]"
severity: major
---
# Independent forward test found final executable ambiguities in task 475
Bacon reconstructed the flow but found the skill still omits the exact landing mutation `dacli project show <slug> --landing-mode pr --landing-base main`, conflicts on canonical logs ordering (help: `dacli logs <run-id-prefix|child-id> [-f] [--tail N]`), does not reconcile `task check <ref> [--n N | --all] [--verify command]`, lacks loop examples with both --impl-role and --review-role, and leaves direct PR merge then owner accept versus multi-task ship ordering implicit. Add these exact current forms, state runtime cooldown has no dedicated clear or inspect command beyond runtime doctor/state, and define retro as `<task-or-project-ref>`. The post-adoption dependency and semantic-dedup limitations are now GitHub #717/#718; the skill should fail safely on affected tasks while still completing an unaffected plan-to-PR scenario.
