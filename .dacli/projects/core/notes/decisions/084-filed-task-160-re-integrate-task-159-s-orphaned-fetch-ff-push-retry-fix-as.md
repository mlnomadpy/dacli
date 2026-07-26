---
id: d-084-filed-task-160-re-integrate-task-159-s-orphaned-fetch-ff-push-retry-fix-as
kind: note
note_kind: decision
created: 2026-07-26T22:10:30Z
created_by: a-y7ksaqj45b
about: [[084]]
---
# 084: filed task 160 (re-integrate task 159's orphaned fetch+ff/push-retry fix) as the single highest-value evidence-based change
## Chose
084: filed task 160 (re-integrate task 159's orphaned fetch+ff/push-retry fix) as the single highest-value evidence-based change
## Rejected
Filing a fresh, unverified code defect surfaced by code review, or re-filing an already-open item (115/117/118)
## Because
Task 159 is marked accepted+completed by a-root with all 5 boxes [x], so the backlog asserts the loop's non-fast-forward defect is FIXED — but its entire implementing commit 8544493 (gitx.FastForward + PushSync, orchestration between-cycle sync, ship.go retry, +315 lines incl. 146 lines of gitx_test.go) lives ONLY on branch dacli/159-...; 'git merge-base --is-ancestor 8544493 main' => NOT ON MAIN, and on main gitx.go has no such helper (Push at gitx.go:219 is still bare), ship.go:168 still surfaces a bare 'push failed', and orchestration.go has no between-cycle ff. This is the exact task-157 orphaned-branch pattern (accepted+closed, branch never integrated) — and task 157's own acceptance told us to sweep for this, yet 159 slipped through. It outranks any fresh defect because it is (1) CONFIRMED live on main via merge-base proof, not an unverified lead; (2) a self-regressing correctness bug in the loop's core git reliability that compounds task 114's now-landed auto-merge path; (3) cheap to fix — the tested fix already exists and only needs re-integration; and (4) actively hidden by a false 'done' in the backlog, so it will never be re-fixed unless re-filed.
