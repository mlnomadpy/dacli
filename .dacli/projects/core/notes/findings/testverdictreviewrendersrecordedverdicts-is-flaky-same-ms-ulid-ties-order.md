---
id: f-testverdictreviewrendersrecordedverdicts-is-flaky-same-ms-ulid-ties-order
kind: note
note_kind: finding
created: 2026-07-22T19:19:11Z
created_by: a-n2nn0v2g31
about: [[073]]
severity: minor
---
# TestVerdictReviewRendersRecordedVerdicts is flaky: same-ms ULID ties order randomly
internal/features/vcs/lifecycle_test.go:101 — the test Appends two EventComment verdicts in the same millisecond and asserts a-seat1 before a-seat2. ulid.New() (internal/ulid/ulid.go:21) is NOT monotonic within a millisecond (80 random bits break ties randomly), so lexicographic=chronological order only holds across ms boundaries. Fails ~50% (incl. under -race). Pre-existing, not caused by task 073. verdictReview() reverses eventlog.List (newest-first) to get chronological order, which is reliable in production (panel votes are seconds apart) but not for same-ms test appends. Fix: make the test appends land in distinct milliseconds (2ms sleep) so it deterministically models real usage; a monotonic-counter fix in ulid would need a process-global lock, contradicting eventlog's documented no-shared-state/no-lock design.
