---
id: f-git-gh-subprocesses-spawn-with-no-context-timeout-a-hung-child-blocks-the-whole-mcp-stdio-server
kind: note
note_kind: finding
created: 2026-07-21T23:09:25Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
---
# git/gh subprocesses spawn with no context/timeout — a hung child blocks the whole MCP stdio server
Every git/gh call is exec.Command(...)+CombinedOutput()/Output() with no context.Context and no deadline: gitx.go:14 (git, incl. push), features/vcs/vcs.go:48, vcs/lifecycle.go:143 (gh pr create), ghmirror.go:38 (gh issue/repo/auth), collab.go:224. The network/auth-bound ones can hang indefinitely (git blocking on a credential-helper prompt, gh on network). Under 'dacli mcp serve' (cli.go:200) a single tools/call that shells out to a hung gh/git blocks the entire stdio Serve loop (mcp.go) with no timeout. Use exec.CommandContext with a deadline.
