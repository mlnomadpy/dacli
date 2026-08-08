---
id: f-267-complete-on-branch-dacli-267-both-acceptance-criteria-met-ready-for-accept
kind: note
note_kind: finding
created: 2026-08-04T14:43:59Z
created_by: a-maintainer-c76h39
about: "[[267]]"
severity: moderate
---
# 267 complete on branch dacli/267-...; both acceptance criteria met, ready for accept
Commit f7dfdbd by a-maintainer-c76h39. (1) spawn/supervise now WARN via warnExeAllowlist in resolveLaunch (execution.go) when os.Executable() is not a path the runtime allowlist permits — backed by new store.RuntimeAllowsDacli(args,exe) (runtimefiles.go) which parses Bash(<path>:*) allowlist rules. (2) Tests cover the mismatch using the real cc-rw shape (absolute-path Bash rule Bash(/Users/tahabsn/go/bin/dacli:*)): TestRuntimeAllowsDacli (store, both split + comma-joined encodings) and TestExeAllowlistWarning (execution). go build/test/vet/gofmt all clean. Owner: accept 267 to check boxes + close, then integrate/merge --task 267.
