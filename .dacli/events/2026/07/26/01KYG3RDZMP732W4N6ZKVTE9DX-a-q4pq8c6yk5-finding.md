---
id: 01KYG3RDZMP732W4N6ZKVTE9DX
kind: event
event_kind: finding
created: 2026-07-26T21:02:08Z
created_by: a-q4pq8c6yk5
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: true
---
Task 154 accepted+closed but its ci.yml change was never merged — SPA vitest/eslint gate is absent on main

Task 154 is done [4/4], all acceptance boxes [x], 'accepted by a-root' + 'completed by a-root' 2026-07-26T20:58:38Z. But its ONLY commit 6e142c9 (adds 'test frontend: npm run test:unit' + 'lint frontend: npm run lint' to ci.yml) is NOT an ancestor of HEAD ('git merge-base --is-ancestor 6e142c9 HEAD' => NOT merged) and has no merge PR ('git log --merges --all | grep 154' => none). Branch dacli/154-... exists locally + on origin, orphaned. CONSEQUENCE: ci.yml on main (.github/workflows/ci.yml:24-36) still runs only 'npm ci && npm run build' — no test:unit, no lint. The 15 frontend test files in internal/features/dashboard/ui/src (incl. DependencyGraph.test.ts guarding task 150, BurnRate.test.ts guarding task 149) never execute in CI, so any regression to the just-merged shadcn dashboard (152) ships green. package.json defines test:unit='vitest run' and lint='eslint . --max-warnings 0'; the 6e142c9 diff applies cleanly onto current ci.yml (npm ci/npm run build context unchanged). This is the 'falsely done' class [[115]] with a concrete, current instance. Remedy: integrate 154's branch (or re-apply the 6-line diff) to main and prove a red frontend test fails CI.
