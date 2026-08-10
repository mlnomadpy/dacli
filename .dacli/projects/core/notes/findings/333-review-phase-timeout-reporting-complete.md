---
id: f-333-review-phase-timeout-reporting-complete
kind: note
note_kind: finding
created: 2026-08-10T17:29:55Z
created_by: a-junior-ymc0q9
about: "[[333]]"
severity: major
---
# 333: review phase timeout reporting complete
All acceptance criteria met:

1. Timeout messages now print 'review spawn timed out (max-tokens: N)' instead of 'spawn refused/failed'
2. Timeout kills ('timed out') and policy refusals ('spawn refused/failed') are distinguishable in output
3. Spawn success banner ('spawning...') is never printed as an error message
4. TestReviewPhaseReportsTimeoutDistinctlyFromRefusal verifies the behavior:
   - Simulates timeout with error containing 'stalled'
   - Asserts log contains 'timeout' but not 'refused'
   - Asserts banner 'spawning' is not printed

Implementation in orchestration.go:reviewPhase():
- Detects timeout by checking err.Error() for 'stalled'
- Reports different messages for timeout vs refusal
- Filters out spawn banner from error output
