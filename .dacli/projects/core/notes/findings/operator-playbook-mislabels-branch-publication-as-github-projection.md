---
id: f-operator-playbook-mislabels-branch-publication-as-github-projection
kind: note
note_kind: finding
created: 2026-08-19T11:55:37Z
created_by: a-fixer-5aj0d0
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: major
---
# Operator playbook mislabels branch publication as GitHub projection
docs/OPERATOR_PLAYBOOK.md:23 and skills/dacli/references/critical-path-github.md:10/17 use github push for a task branch. Current help distinguishes dacli push <ref> (task branch) from dacli github push <project> [task-ref...] (issue projection); source confirms top-level next accepts --project/--parallel in internal/features/insight/insight.go:152.
