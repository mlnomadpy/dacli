---
id: t-01M0F8DMG5NA6198RGF59NKXWC
kind: task
created: 2026-08-20T09:35:47Z
created_by: a-root
owner: a-root
github:
  issue: 760
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# [agent-report] github sync --dry-run mutates by adopting issues before rejecting the flag
## Context
Adopted from GitHub issue #760.

Running 'dacli github sync bashnota --dry-run' on a linked public repository adopted 12 GitHub issues as tasks, printed 'pull: 12 adopted', and only afterward failed with 'unknown flag(s): --dry-run'. The shipped GitHub landing documentation explicitly recommends this preview command. Expected: dry-run performs no local or remote mutations, or rejects unsupported flags before any operation. Actual: inbound adoption mutated dacli state before validation failure. Please validate flags before starting sync and add a regression that snapshots task/event state around sync --dry-run.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] `dacli github sync <project> --dry-run` validates the complete shared flag set before either the pull or push half can mutate local or remote state.
- [ ] A valid dry-run previews both inbound adoptions and outbound task/decision/finding effects while leaving tasks, events, mappings, notes, GitHub issues, and comments unchanged.
- [ ] An unknown or conflicting flag fails with exit 2 before any GitHub request that can write and before any local workspace write.
- [ ] The real non-dry-run sync still performs pull then push, including mirroring a freshly adopted issue in the same invocation.
- [ ] Shared sync flags such as task refs, `--since`, `--findings-as-issues`, and `--with-tasks` retain their documented forwarding behavior.
- [ ] Tests snapshot task/event/mapping state and fake remote writes around public `github sync --dry-run`; mutation proof shows deferring validation until after pull makes the regression fail.
- [ ] CLI/MCP help and GitHub/operator documentation derive or validate the exact preview signature.
- [ ] `gofmt -l .`, `go vet ./...`, pinned `golangci-lint run`, and `go test ./...` pass.
## Log
