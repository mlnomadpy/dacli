---
id: t-01KY7RXS7K5MDC36QTY9H6R3B5
kind: task
created: 2026-07-23T15:18:54Z
created_by: a-qy5e8fvxm5
owner: a-root
priority: should
---
# Give the last three unbounded git and gh subprocesses deadlines so gitx's every-git-child invariant holds
## So that
gitx.go:15-19 claims deadlines bound every git child as a correctness property, but tasks 018 and 105 left three call sites unbounded (105 was scoped strictly to orchestration driver.git); two are network ops that hang on a wedged network or credential prompt and under dacli mcp serve freeze the whole stdio loop
## Acceptance
- [x] internal/skills/skills.go:170 git clone --depth 1 runs under the network deadline (gitx.RunNetwork or exec.CommandContext with gitx.NetworkTimeout) instead of a bare exec.Command, so dacli skill install aborts a hung clone instead of blocking forever
- [x] internal/features/collab/collab.go:236 gh issue create (the escalate --github path) runs under a context deadline via exec.CommandContext, matching selfreport.go:108 and vcs/lifecycle.go:47 and ghmirror.go:58; the escalation event still stands on timeout
- [x] internal/store/version.go lines 98,131,139,148 (FileChangelog and VersionIsStale) run their four git calls under the local deadline (gitx.Run or exec.CommandContext with gitx.LocalTimeout), degrading to the existing no-history fallback on timeout
- [x] grep for exec.Command git and exec.Command gh across internal excluding _test.go returns zero non-deadline call sites
- [x] go build ./... clean and go test ./internal/... green with DACLI_AGENT cleared
## Log
- 2026-07-23T16:00:21Z claimed by a-jr8aj4x65b
- 2026-07-23T16:04:10Z adopted by a-root (owner a-qy5e8fvxm5 orphaned)
- 2026-07-23T16:04:10Z accepted by a-root
- 2026-07-23T16:04:10Z completed by a-root
