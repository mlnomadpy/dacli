---
id: d-issuecomments-returns-string-error-mirrorfindings-skips-the-task-on-a-read
kind: note
note_kind: decision
created: 2026-08-04T12:06:58Z
created_by: a-maintainer-bqv5pa
about: "[[220]]"
---
# issueComments returns ([]string, error); mirrorFindings skips the task on a read/parse failure
## Chose
issueComments returns ([]string, error); mirrorFindings skips the task on a read/parse failure
## Rejected
keep the []string signature and return an empty slice on parse failure, or post anyway on failure
## Because
an empty slice is indistinguishable from 'issue genuinely has no comments' — the idempotency check then treats an unreadable list as 'nothing posted' and re-posts every finding, duplicating comments on each push (the reported bug). An error return lets mirrorFindings fail closed (post nothing, retry next push) while an empty-but-successful read still allows the legitimate first post.
