---
id: f-github-issue-acceptance-criteria-must-be-corrected-before-adoption
kind: note
note_kind: finding
created: 2026-08-19T12:20:05Z
created_by: a-maintainer-68b2n1
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: moderate
---
# GitHub issue acceptance criteria must be corrected before adoption
A neutral forward test found that github pull imports a GitHub issue's ## Acceptance criteria checkbox list, while the documented shipped CLI has no task-edit command to add missing acceptance criteria after adoption. docs/OPERATOR_PLAYBOOK.md and skills/dacli/references/critical-path-github.md now require checkable criteria before pull.
