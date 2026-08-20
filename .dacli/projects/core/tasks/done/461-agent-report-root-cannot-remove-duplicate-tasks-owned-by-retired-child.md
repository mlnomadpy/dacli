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
- [x] A public-command regression spawns a child, creates a child-owned duplicate task, retires the child, and proves read-write root can remove it.
- [x] The regression resolves the child through the same retired-agent storage/layout used by the real `agent retire` command, not an in-memory shortcut.
- [x] A live child owner still causes exit 3 and the refusal names the owner and live-run condition.
- [x] A non-root sibling and read-only root-shaped identity still receive exit 3.
- [x] Unknown/non-child owners remain non-removable through the root orphan exception.
- [x] `--force` remains required for history-bearing, active, done, or blocked tasks and never bypasses task references.
- [x] Mutation evidence demonstrates the retired-child regression fails when retired identities are excluded from lifecycle lookup.
- [x] Focused planning/store tests and `go test ./...` pass.
## Log
- 2026-08-18T14:38:46Z claimed by a-maintainer-w5nkdg
- 2026-08-19T11:40:57Z accepted by a-root
- 2026-08-19T11:40:58Z verified by `GOCACHE=/tmp/dacli-461-accept go test ./...` (exit 0) in branch main at 1eb453e — proves that tree builds, not that the work is in trunk
- 2026-08-19T11:40:58Z deliverable: dacli/461-agent-report-root-cannot-remove-duplicate-tasks-owned-by-retired-child is merged into main
- 2026-08-19T11:40:58Z completed by a-root
- 2026-08-19T11:48:59Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/705 (event 01M0CWK85DBG1X2HVJF77B92Y4)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-461-accept go test ./...","exit_code":0,"duration_ms":127658,"artifact_hash":"sha256:3fb61e358d9e85171fa419218ca6bc9fa7aa64184ab10a4ecf2d25617b926d44","verifier":"a-root","branch":"main","commit_sha":"1eb453eff30d7e3a803c28365f151538997d5a11"}
{"command":"GOCACHE=/tmp/dacli-461-accept go test ./...","exit_code":0,"duration_ms":2916,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"main","commit_sha":"1eb453eff30d7e3a803c28365f151538997d5a11"}
