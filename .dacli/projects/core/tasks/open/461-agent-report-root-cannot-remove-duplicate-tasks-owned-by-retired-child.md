---
id: t-01M088WV1WEBW031R2046WVZSW
kind: task
created: 2026-08-17T16:29:24Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 684
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# [agent-report] root cannot remove duplicate tasks owned by retired child
## Context
Adopted from GitHub issue #684.

As `dacli whoami` reports a-root with grant rw, `dacli task rm 057` and `task rm 059` both refuse: owned by a-reviewer-fep2a4 — only its owner or root can remove it. `--force`, root claim plus sync, and retiring a-reviewer-fep2a4 do not change the refusal. The two tasks are exact duplicates of root-owned 049 and 051, so root needs a supported reconciliation/removal path.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] A public-command regression spawns a child, creates a child-owned duplicate task, retires the child, and proves read-write root can remove it.
- [ ] The regression resolves the child through the same retired-agent storage/layout used by the real `agent retire` command, not an in-memory shortcut.
- [ ] A live child owner still causes exit 3 and the refusal names the owner and live-run condition.
- [ ] A non-root sibling and read-only root-shaped identity still receive exit 3.
- [ ] Unknown/non-child owners remain non-removable through the root orphan exception.
- [ ] `--force` remains required for history-bearing, active, done, or blocked tasks and never bypasses task references.
- [ ] Mutation evidence demonstrates the retired-child regression fails when retired identities are excluded from lifecycle lookup.
- [ ] Focused planning/store tests and `go test ./...` pass.
## Log
