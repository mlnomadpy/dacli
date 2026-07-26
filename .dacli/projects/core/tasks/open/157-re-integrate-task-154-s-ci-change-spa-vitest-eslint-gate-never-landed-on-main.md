---
id: t-01KYG3RQD8T8JJVM1VAPMW40KH
kind: task
created: 2026-07-26T21:02:18Z
created_by: a-q4pq8c6yk5
owner: a-q4pq8c6yk5
priority: should
---
# Re-integrate task 154's CI change: SPA vitest+eslint gate never landed on main (accepted+closed but branch orphaned)
## Acceptance
- [ ] ci.yml 'test' job on main runs 'npm run test:unit' AND 'npm run lint' in internal/features/dashboard/ui after the frontend build (re-apply commit 6e142c9's 6-line diff / integrate branch dacli/154-...)
- [ ] A deliberately-broken frontend test assertion makes the CI test job go red (not green), proving the 15 SPA test files now gate on main
- [ ] Verify no other accepted+closed task branch is similarly orphaned off main (spot-check recent done tasks with 'git merge-base --is-ancestor <task-commit> HEAD'); if the loop's accept-close ran without merging, link to [[115]] rather than re-fixing the loop here
## Log
