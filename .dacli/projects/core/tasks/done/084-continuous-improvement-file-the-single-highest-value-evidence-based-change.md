---
id: t-01KY60QM1Y7DK05WXB954YNDHJ
kind: task
created: 2026-07-22T22:56:52Z
created_by: loop
owner: a-root
priority: should
github:
  issue: 187
  repo: mlnomadpy/dacli
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# Continuous improvement: file the single highest-value evidence-based change
## Context
Standing anchor for the autonomous review phase. Survey the code, tests, CI, and open findings; identify the ONE highest-value improvement grounded in evidence (a failing test, a reviewer finding, a real defect); `dacli task add` it with concrete acceptance criteria. Do NOT implement it here, and do NOT invent speculative work.
## Acceptance
- [ ] Filed at least one new task grounded in an observed defect, finding, or failing check
- [ ] Did not implement any change in this task
## Log
- 2026-07-22T22:56:52Z claimed by a-2jxa2ck7jh
- 2026-08-03T22:44:12Z adopted by a-root (owner loop orphaned)
- 2026-08-04T00:37:40Z finding by a-2jxa2ck7jh: loop progress metric counts accept-close, not trunk merges, under --pr --auto (event 01KY60ZRZ1M04NG6SXC11X47SZ)
- 2026-08-04T00:37:40Z finding by a-7ahpwv8p4f: loop Idle path is ungoverned: unbounded review-spawns, thrash-halt bypassed, non-yolo never checkpoints on empty backlog (event 01KY61GVQZNZ5N4CKQC5R75XD9)
- 2026-08-04T00:37:40Z finding by a-48ab0df8g5: onboard TODO-scanner matches marker names as bare substrings in string literals and comments, polluting every brief's codebase map (event 01KY64CMMMAVE92AAY2NSGVRVQ)
- 2026-08-04T00:37:40Z finding by a-yf43x5hk79: Stored codebase map is stale: 13 phantom 'Open markers' reach every brief (fixed by task 087's scanner but never regenerated) (event 01KY78TMTX4EPSZ2PCQD4RXDYB)
- 2026-08-04T00:37:40Z status done proposed by a-yf43x5hk79, applied (event 01KY78TSFJR7T00GSHY4500PDZ)
- 2026-08-04T00:37:40Z finding by a-534c4gav5p: loop --pr force-closes spawn-refused tasks, silently losing them from the backlog (event 01KY79EJHTFYWVCEAJZFES1X5S)
- 2026-08-04T00:37:40Z finding by a-waq3de2hcs: loop BUILD phase picks tasks by seq number, ignoring MoSCoW priority and critical path (event 01KY7E6B4J2BSW4J03R6M2A4R6)
- 2026-08-04T00:37:40Z status done proposed by a-waq3de2hcs, applied (event 01KY7E7BKEA3G93F1RTW8NJQHW)
- 2026-08-04T00:37:40Z finding by a-m146x20e8d: github pull imports human-CLOSED issues as fresh open tasks (State ignored) (event 01KY7ENAK9H079DRTASZTD401D)
- 2026-08-04T00:37:40Z status done proposed by a-m146x20e8d, applied (event 01KY7EP08T65TS7TK5WYVBDGS0)
- 2026-08-04T00:37:40Z finding by a-g3ya9r93e3: Perpetual loop's git subprocesses have no deadline; a hung 'git fetch origin' freezes the whole loop (event 01KY7FY3GZVCJNFQ32HANYHPBC)
- 2026-08-04T00:37:40Z status done proposed by a-g3ya9r93e3, applied (event 01KY7FZ6JYT9PSGK7QSP38WPDH)
- 2026-08-04T00:37:40Z finding by a-0b77j7k11m: loop idle-cycle review spawns are never charged to the token window, defeating --window-tokens in the loop's steady state (event 01KY7J6GP50MG7EGB308N6CYYA)
- 2026-08-04T00:37:40Z finding by a-8p0kde6tvt: Sibling finding f-waq3de2hcs (loop BUILD picks tasks by seq, ignoring MoSCoW/CPM) is now STALE — already fixed (event 01KY7KSSFJ814Y080N12WMJ5N5)
- 2026-08-04T00:37:40Z finding by a-qy5e8fvxm5: Three git/gh subprocesses still unbounded; gitx's 'every git child' deadline invariant is false in the current tree (event 01KY7RTB3ZCD7F8F3W6DNZE8NW)
- 2026-08-04T00:37:40Z finding by a-92n83ap1x9: mdstore has quote-aware GetList read but no SetList write; gates.go:313 is an unguarded duplicate of the antipattern task 111 fixed in runtimefiles (event 01KY84RZT4DM8BJN54671QWSW9)
- 2026-08-04T00:37:40Z finding by a-nfazzjdrh2: mdstore SetList round-trip is lossy for elements holding both quote chars plus a comma (event 01KY85KC0R0KMJN9GFXR7J20YT)
- 2026-08-04T00:37:40Z finding by a-zq4qdv7py6: github-pull 'closed-issue import' bug is refuted: shouldImport already skips closed+unmapped issues and it is unit-tested (event 01KY88VC9PPBT7S40RX4AC5MGE)
- 2026-08-04T00:37:40Z finding by a-kbypt902fn: loop status: window_tokens field holds spent, not the ceiling; budget ceiling never persisted (event 01KY9X893C55EHW31HKEZ07KQG)
- 2026-08-04T00:37:40Z finding by a-q2y900ts5s: burn alert dilutes per-run rate: Series counts all runs, Ceiling only completing runs (false-negative yell) (event 01KYFGJ0CTBXQ1X7T3PA6DKWBX)
- 2026-08-04T00:37:40Z finding by a-avy9rqtfdw: DAG view highlights non-critical edges: uses from.critical && to.critical, not adjacency on critical_path (event 01KYFHQTR8RYQHCSQDG8BNNB2J)
- 2026-08-04T00:37:40Z finding by a-1hwz5pcjva: burn Rate population filter is defeated by any ro-grant role missing role_kind (reviewer.md) (event 01KYFP68109BKE3PCP2ZFE79R9)
- 2026-08-04T00:37:40Z finding by a-hxr220kqc4: CI builds the SPA but never runs its vitest suite — 14 frontend tests are dead weight in CI (event 01KYFQ5TDY9D6JAEWBMGF5WST4)
- 2026-08-04T00:37:40Z finding by a-q4pq8c6yk5: Task 154 accepted+closed but its ci.yml change was never merged — SPA vitest/eslint gate is absent on main (event 01KYG3RDZMP732W4N6ZKVTE9DX)
- 2026-08-04T00:37:40Z finding by a-s4764r5zf3: Verified in source: loop never fetches/ff local main after --auto PR merges; next push fails non-fast-forward (filed task 159) (event 01KYG6VCXKVCDB4JR49V9YS41K)
- 2026-08-04T00:37:40Z finding by a-y7ksaqj45b: Task 159's fetch+ff/push-retry fix is accepted+done but orphaned off main — the non-fast-forward defect is still live (event 01KYG7KNM6M5GDK1KXSD06Y8DQ)
- 2026-08-04T00:37:40Z finding by a-vav46gnkax: loop --pr pendingAccept/pendingLand not persisted: bounded/restarted loop never closes merged tasks and re-opens duplicate PRs (event 01KYGA8JMKJRWQNAJTZNR399SH)
- 2026-08-04T00:37:40Z finding by a-ham9kmbg3b: burn Rate counts non-completing implementer runs the Ceiling excludes, re-opening the 149/153 false-negative on a new axis (event 01KYGAVA926NR8HJ0KV87SGRWE)
