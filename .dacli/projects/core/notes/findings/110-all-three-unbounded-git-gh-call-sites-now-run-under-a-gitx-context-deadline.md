---
id: f-110-all-three-unbounded-git-gh-call-sites-now-run-under-a-gitx-context-deadline
kind: note
note_kind: finding
created: 2026-07-23T16:03:49Z
created_by: a-jr8aj4x65b
about: [[110]]
severity: moderate
---
# 110: all three unbounded git/gh call sites now run under a gitx/context deadline
internal/skills/skills.go:171 git clone now runs via gitx.RunNetwork (120s deadline) instead of a bare exec.Command, so dacli skill install/Fetch aborts a hung clone. internal/features/collab/collab.go:239 gh issue create (escalate --github) now runs under exec.CommandContext with a 120s context.WithTimeout, matching selfreport.go and vcs/lifecycle.go's runGH/ghmirror.go pattern; the prior EventHelp escalation event still stands on timeout, only the gh mirror times out. internal/store/version.go FileChangelog (git log --follow) and VersionIsStale (git status --porcelain, git log -S, git log range) now run via gitx.Run (30s local deadline) instead of exec.Command with -C dir; behavior preserved (git status/log semantics unchanged, absolute path for FileChangelog's --, base-relative pathspec + cmd.Dir for VersionIsStale). grep -rn 'exec\.Command(' internal --include='*.go' | grep -v _test.go | grep -E '"git"|"gh"' returns zero hits. go build ./... clean; go test ./internal/... all green (33 packages, no failures).
