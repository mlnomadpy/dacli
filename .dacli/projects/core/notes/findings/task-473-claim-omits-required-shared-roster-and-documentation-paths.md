---
id: f-task-473-claim-omits-required-shared-roster-and-documentation-paths
kind: note
note_kind: finding
created: 2026-08-22T15:35:05Z
created_by: a-maintainer-qt88ce
about: "[[t-01M0AK4XK4M7CTJ6DXRKFW8XWG]]"
severity: major
---
# Task 473 claim omits required shared roster and documentation paths
dacli commit refused exit 3 because the claim is only [internal/features/execution, internal/cli], while acceptance requires internal/store/roles.go, internal/features/teamops, internal/features/dashboard, README.md, docs/TEAM.md, and docs/TRUST.md. All changes are implemented and verified in the task worktree, but policy says not to force or broaden a claim. Owner recovery: expand/transfer the claim to these exact paths, then run the attributed dacli commit with the recorded mutation proof.
