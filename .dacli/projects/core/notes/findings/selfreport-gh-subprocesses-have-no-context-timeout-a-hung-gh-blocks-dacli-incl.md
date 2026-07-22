---
id: f-selfreport-gh-subprocesses-have-no-context-timeout-a-hung-gh-blocks-dacli-incl
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-9y38s7w8e2
about: [[t-01KY59FNFK27A1084PQ8R2CJ5S]]
source_event: 01KY59RD05V5AY73TPTDRP41WC
---
# selfreport gh subprocesses have no context timeout — a hung gh blocks dacli (incl. mcp serve)
selfreport.go:74 (exec.Command('gh','auth','status')) and selfreport.go:78 (exec.Command('gh','issue','create',...)) use bare exec.Command + CombinedOutput/Output with no context.Context or deadline. Both are network/auth-bound and can hang indefinitely (credential prompt, dead network). ghmirror.go:44 already fixed this class by wrapping gh in context.WithTimeout(120s); selfreport was not updated and is a remaining instance of the sibling finding f-git-gh-subprocesses-spawn-with-no-context-timeout. Under 'dacli mcp serve' a hung 'dacli report' blocks the stdio loop. Fix: mirror ghmirror's exec.CommandContext + WithTimeout wrapper.
