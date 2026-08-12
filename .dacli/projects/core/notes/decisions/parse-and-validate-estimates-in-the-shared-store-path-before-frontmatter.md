---
id: d-parse-and-validate-estimates-in-the-shared-store-path-before-frontmatter
kind: note
note_kind: decision
created: 2026-08-12T16:58:26Z
created_by: a-codex-maintainer-gqkrc4
about: "[[381]]"
github:
  issue: 500
  repo: mlnomadpy/dacli
---
# Parse and validate estimates in the shared store path before frontmatter mutation
## Chose
Parse and validate estimates in the shared store path before frontmatter mutation
## Rejected
Add separate string checks in task add and task estimate handlers
## Because
CreateTask and SetEstimate share setEstimateFront; central numeric parsing plus ThreePoint.Valid keeps every persistence caller consistent, while task add performs the same validation early only to preserve usage exit code 2.
