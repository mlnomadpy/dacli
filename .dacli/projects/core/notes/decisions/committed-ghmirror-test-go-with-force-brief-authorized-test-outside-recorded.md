---
id: d-committed-ghmirror-test-go-with-force-brief-authorized-test-outside-recorded
kind: note
note_kind: decision
created: 2026-07-22T18:28:32Z
created_by: a-n03m4hw62x
about: [[066]]
---
# committed ghmirror_test.go with --force (brief-authorized test outside recorded claim)
## Chose
committed ghmirror_test.go with --force (brief-authorized test outside recorded claim)
## Rejected
drop the unit tests to stay inside the ghmirror.go-only claim
## Because
the acceptance criterion REQUIRES unit tests on fixtures (marker/idempotency/label-mapping, no live gh); the test file is the same slice and load-bearing for acceptance, so --force with a note is correct
