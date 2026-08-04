---
id: f-task-159-s-fetch-ff-push-retry-fix-is-accepted-done-but-orphaned-off-main-the
kind: note
note_kind: finding
created: 2026-08-04T00:37:40Z
created_by: a-y7ksaqj45b
about: "[[t-01KY60QM1Y7DK05WXB954YNDHJ]]"
source_event: 01KYG7KNM6M5GDK1KXSD06Y8DQ
---
# Task 159's fetch+ff/push-retry fix is accepted+done but orphaned off main — the non-fast-forward defect is still live
Task 159 (loop fetch+ff local main / ship --push retry on non-fast-forward) is marked accepted+completed by a-root with all 5 acceptance boxes [x]. But its implementing commit 8544493 ('159: add gitx.FastForward + PushSync, sync local trunk between loop cycles, retry push on non-fast-forward', +315 across 6 files incl. 146 lines of gitx_test.go) exists ONLY on branch dacli/159-...; 'git merge-base --is-ancestor 8544493 main' => NOT ON MAIN. On main today: (1) internal/gitx/gitx.go has NO FastForward/PushSync/Sync/Fetch helper — Push (gitx.go:219) is still a bare 'push -u origin <branch>'; (2) ship.go:168 still returns a bare 'push failed: <out>' with NO fetch+ff retry on non-fast-forward; (3) orchestration.go has NO between-cycle fetch+ff — the only ff reconciliation on main is vcs/lifecycle.go:815 'pull --ff-only' which fires solely on the synchronous gated gh-merge path, never on --auto. So once task 114's deferred-record-push lets --auto merges land under branch protection, local main diverges behind origin and the next ship --push fails non-fast-forward, error only best-effort logged. This is the task-157 orphaned-branch pattern (accepted+closed but never integrated). The backlog FALSELY reports 159 done, so it will not be re-fixed unless re-filed.
