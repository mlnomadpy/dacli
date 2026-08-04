---
id: f-220-complete-on-branch-dacli-220-issuecomments-parse-failure-now-fails-closed
kind: note
note_kind: finding
created: 2026-08-04T12:07:19Z
created_by: a-maintainer-bqv5pa
about: "[[220]]"
severity: moderate
---
# 220 complete on branch dacli/220-...: issueComments parse failure now fails closed, no longer reposts finding comments
Commit 66069d7 by a-maintainer-bqv5pa. Root cause: issueComments (ghmirror.go:709) returned nil on BOTH a gh fetch error AND a json.Unmarshal parse failure. mirrorFindings (ghmirror.go:752) then treated the empty slice as 'issue has no comments', so commentsHaveMarker returned false for every finding and it re-posted ALL of them via 'gh issue comment' on the next push — a transient JSON/gh hiccup duplicated every finding comment.

Fix: issueComments now returns ([]string, error), wrapping the parse failure (fmt.Errorf 'parse issue %d comments'). mirrorFindings checks the error and returns 0 (posts nothing) when the existing-comment list is unreadable, failing closed and retrying on the next push; a genuinely-empty successful read still allows the legitimate first post.

Acceptance [1/1] met: 'an unparseable comment list is a failure not an empty list' — TestIssueCommentsParseFailureIsError asserts the error return; TestMirrorFindingsSkipsOnUnreadableComments asserts 0 comments posted on an unreadable read.

Verified by reproduction: temporarily re-swallowing the parse error (return nil,nil) made TestMirrorFindingsSkipsOnUnreadableComments fail with 'posted 1 comment(s) ... want 0' (the exact bug); restoring the error return made both green. go build ./... clean, go vet clean, gofmt -l internal/ clean, go test ./internal/features/ghmirror/ ok. Full go test ./... green except the pre-existing DACLI_AGENT env-leak in internal/features/catalog (green under 'go test -exec env -u DACLI_AGENT'), unrelated to this change. Owner: accept 220 to check the box + mark done, then integrate/merge --task 220.
