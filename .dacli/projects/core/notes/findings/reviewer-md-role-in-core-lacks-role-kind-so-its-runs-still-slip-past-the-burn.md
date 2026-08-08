---
id: f-reviewer-md-role-in-core-lacks-role-kind-so-its-runs-still-slip-past-the-burn
kind: note
note_kind: finding
created: 2026-07-26T16:51:53Z
created_by: a-w190nhae40
about: [[149]]
severity: minor
---
# reviewer.md role in core lacks role_kind, so its runs still slip past the burn Rate implementer filter
The fix filters review-role runs by team.Role.Kind=="reviewer" (reviewerRoles in internal/features/dashboard/burn.go). The built-in roster (internal/features/wscore/rosters.go:26) sets Kind:"reviewer", and .dacli/roles/frontend-reviewer.md declares role_kind: reviewer -- both are caught. But .dacli/roles/reviewer.md (created 2026-07-21) has NO role_kind field, so a run with role: reviewer is NOT excluded in THIS workspace. Verify-panel seats (the larger diluting population) are still caught by their explicit verify_panel_seat marker regardless of role. Fix is data-only: backfill 'role_kind: reviewer' into .dacli/roles/reviewer.md (out of this task's code scope).
