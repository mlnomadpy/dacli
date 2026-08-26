---
id: f-task-490-concurrent-policy-updates-serialized
kind: note
note_kind: finding
created: 2026-08-26T15:24:00Z
created_by: a-root
about: "[[490]]"
severity: major
---
# Task 490 concurrent policy updates serialized
Third PR #795 re-review reproduced two successful one-flag updates losing one value. UpdateProjectLanding now holds .project.lock across load/resolve/save/reload. TestConcurrentOneFlagProjectShowUpdatesSerialize holds that lock around two real CLI subprocesses, proves neither bypasses it, then verifies mode=pr and base=release persist together. Mutating the implementation to use a different lock makes the test fail: project show bypassed the project transaction lock.
