---
id: f-direct-next-invocation-supersedes-its-misleading-help-alias
kind: note
note_kind: finding
created: 2026-08-19T11:51:23Z
created_by: a-fixer-cts0zq
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: moderate
---
# Direct next invocation supersedes its misleading help alias
Although  renders queue-next usage, the actual  run succeeded and returned a ready task, while  returned not found. Canonical docs retain next --project/--parallel; only the invalid push --task form was corrected to github push <project> <ref>.
