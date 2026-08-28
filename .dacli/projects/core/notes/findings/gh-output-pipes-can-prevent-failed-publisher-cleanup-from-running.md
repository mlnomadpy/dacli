---
id: f-gh-output-pipes-can-prevent-failed-publisher-cleanup-from-running
kind: note
note_kind: finding
created: 2026-08-28T09:06:38Z
created_by: a-maintainer-fb7kb1
about: "[[t-01M1068MTFPQ6YFVQG204M2EX4]]"
severity: major
---
# gh output pipes can prevent failed publisher cleanup from running
internal/features/ghmirror/ghmirror.go: mutating gh commands previously relied on os/exec-managed writer pipes; when the gh leader exits but a forked publisher retains stdout/stderr, Cmd.Wait blocks before tree cleanup and the github-push lease cannot be released. Deterministically reproduced by the offline fork fixture in process_tree_test.go.
