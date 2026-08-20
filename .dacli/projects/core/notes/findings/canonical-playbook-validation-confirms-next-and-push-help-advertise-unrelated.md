---
id: f-canonical-playbook-validation-confirms-next-and-push-help-advertise-unrelated
kind: note
note_kind: finding
created: 2026-08-19T12:18:36Z
created_by: a-maintainer-68b2n1
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: major
---
# Canonical playbook validation confirms next and push help advertise unrelated signatures
Fresh validation on 2026-08-19: /tmp/dacli-current-bin next --help prints 'dacli queue next <slug>' while internal/features/insight/insight.go implements the top-level scheduler next command with --project/--parallel; /tmp/dacli-current-bin push --help prints 'dacli github push <project> ...' while internal/features/vcs/lifecycle.go implements task branch push. Task 475 docs use the actual scheduler-next and branch-push forms; fixing internal Usage is outside this documentation correction scope.
