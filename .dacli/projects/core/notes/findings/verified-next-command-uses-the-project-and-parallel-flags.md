---
id: f-verified-next-command-uses-the-project-and-parallel-flags
kind: note
note_kind: finding
created: 2026-08-19T11:51:29Z
created_by: a-fixer-cts0zq
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: moderate
---
# Verified next command uses the project and parallel flags
Direct run: /tmp/dacli-current-bin next --project core --parallel 1 returned the current ready task. The queue next core form returned not found: queue/core. Canonical playbook and skill references therefore keep next --project <project> --parallel <width>.
