---
id: f-added-deterministic-idle-review-build-integration-test
kind: note
note_kind: finding
created: 2026-07-23T12:23:52Z
created_by: a-jcc2s9x0rh
about: [[097]]
severity: minor
---
# Added deterministic idle->review->build integration test
internal/features/orchestration/driver_test.go: new filingRunner (embeds fakeRunner, records calls but executes none) simulates the review phase's real-world side effect — filing a task — by calling store.CreateTask on the first spawn carrying the review role. New TestDriverIdleReviewFilesTaskThenBuilds drives an empty backlog through: Idle decision (backlog=0) -> reviewPhase spawns go-auditor -> filingRunner files a task -> loop continues (non-dry-run, sleep is a no-op) -> next Before() sees ready backlog=1 -> Proceed -> runCycle spawns fixer build targeting exactly the filed task's ref. Asserts (1) first spawn overall is the review role, not a builder; (2) a fixer build spawn follows targeting --task <filedRef>. No real process is ever spawned (runner is the fake throughout); go test -race -count=3 ./internal/features/orchestration/... all green.
