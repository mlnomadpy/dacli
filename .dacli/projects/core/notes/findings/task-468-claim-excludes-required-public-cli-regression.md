---
id: f-task-468-claim-excludes-required-public-cli-regression
kind: note
note_kind: finding
created: 2026-08-18T13:01:52Z
created_by: a-maintainer-ytrsg6
about: "[[t-01M0AETPE835JWHHS5GA5RE4AW]]"
severity: major
---
# Task 468 claim excludes required public CLI regression
dacli commit refused exit 3 because the live claim allows internal/store and internal/features/teamops but excludes internal/cli/spawn_test.go. The acceptance-required real mock-runtime spawn -> terminal non-retired identity -> role rm -> agent show/runs show regression was implemented and passed there, but was removed to honor slice isolation. Manual recovery: widen/respawn the claim to include internal/cli/spawn_test.go and restore that public arc; do not use --force.
