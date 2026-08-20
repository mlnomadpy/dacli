---
id: f-operator-playbook-command-validation-exposed-stale-queue-and-push-forms
kind: note
note_kind: finding
created: 2026-08-19T11:50:07Z
created_by: a-fixer-cts0zq
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: major
---
# Operator playbook command validation exposed stale queue and push forms
Current help reports queue next <slug> and github push <project> [task-ref...]; the new playbook/references had documented next --parallel and push --task. Corrected in docs/OPERATOR_PLAYBOOK.md and skills/dacli/references/*.md, with docs/support_claims_test.go covering the recovery-safe retro and workspace reference wording.
