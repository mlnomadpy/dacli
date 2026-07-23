---
id: f-099-complete-agents-tail-text-runtime-buffering-note
kind: note
note_kind: finding
created: 2026-07-23T12:43:47Z
created_by: a-wrwzzt98sh
about: [[099]]
severity: minor
---
# 099 complete: agents --tail text-runtime buffering note
Commit b29cc05 by a-wrwzzt98sh. Staged only internal/features/execution/{execution.go,stream_test.go} via git add + dacli commit --no-add. ACCEPTANCE: (1) when a live agent's runtime has no usage_format (a text runtime whose child fully-buffers stdout until exit) and transcript.log has no renderable line yet, 'dacli agents --tail' now prints '(text runtime — output appears at exit)' instead of the generic '(no transcript output yet)'; stream-json runtimes and any runtime that already produced output are unaffected (execution.go: new tailLine()/isTextRuntime() helpers around execution.go:1324, memoized per invocation via a runtime-name->bool cache so store.LoadRuntime is called at most once per distinct runtime in the live list). (2) covered by TestTailLineDistinguishesTextRuntimeFromNoOutputYet in stream_test.go, which exercises all 4 branches: text runtime+no output, stream-json runtime+no output, unresolvable runtime name (falls back to generic message, not misread as text), and a transcript with real content (always wins regardless of runtime). go build ./... clean; go test -exec 'env -u DACLI_AGENT' ./internal/... all green. Owner: verify and close via dacli task check/done + dacli merge --task 099.
