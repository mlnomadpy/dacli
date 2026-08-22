---
id: f-claude-authentication-probe-prevents-governed-run-creation
kind: note
note_kind: finding
created: 2026-08-22T22:00:11Z
created_by: a-fixer-76xc6t
about: "[[t-01M0CZANEM3TFEMGTW3NTNXGXM]]"
severity: major
---
# Claude authentication probe prevents governed run creation
internal/features/execution/behavioral_preflight.go:59 declares claude-print-v1; preflight_test.go:20 uses a version-capable fake Claude that emits Not logged in · Please run /login. Doctor reports launch incompatible/authentication with the /login remedy, then preflight and spawn return exit 3 before task-path mutation. Mutation: removing claude-print-v1 from hasBehavioralPreflight made focused test fail at preflight_test.go:41.
