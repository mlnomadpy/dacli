---
id: d-use-scheduler-next-and-branch-push-forms-in-the-canonical-playbook
kind: note
note_kind: decision
created: 2026-08-19T11:55:37Z
created_by: a-fixer-5aj0d0
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
github:
  issue: 731
  repo: mlnomadpy/dacli
---
# Use scheduler next and branch push forms in the canonical playbook
## Chose
Use scheduler next and branch push forms in the canonical playbook
## Rejected
Document queue next or github push as interchangeable forms
## Because
The insight scheduler parses --project/--parallel while queue next reads queue steps, and the vcs push command publishes the branch whereas ghmirror push projects issue state.
