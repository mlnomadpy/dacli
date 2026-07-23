---
id: f-setinline-now-quotes-comma-bracket-brace-quote-containing-elements
kind: note
note_kind: finding
created: 2026-07-23T18:41:10Z
created_by: a-v05q6gkkqh
about: [[111]]
severity: minor
---
# setInline now quotes comma/bracket/brace/quote-containing elements
internal/store/runtimefiles.go: added quoteListElem(), used by setInline for invoke_args/sandbox_ro_args/env_passthrough. An element containing a comma, [, ], {, }, #, a quote char, or leading/trailing whitespace is wrapped in double quotes (or single quotes if it itself contains a double quote but no single quote). Proven by new TestRuntimeInlineListRoundTripsCommaContainingElements in internal/store/runtimefiles_test.go, which round-trips the claude-code preset's SandboxRO=[--allowedTools, Read,Grep,Glob,LS,Bash(dacli:*)] through CreateRuntime+LoadRuntime and asserts element-for-element equality. go build ./... and go test ./internal/... both green.
