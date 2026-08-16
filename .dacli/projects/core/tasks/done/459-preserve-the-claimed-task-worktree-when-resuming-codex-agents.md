---
id: t-01KZZVWMD0KYAPDN9QMQDK1GF3
kind: task
created: 2026-08-14T10:08:10Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 673
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Preserve the claimed task worktree when resuming Codex agents
## Context
Adopted from GitHub issue #673.

In azettaai/perio task 044, dacli spawn --task 044 --role fixer --runtime codex-terra-rw --worktree --detach --pr correctly created .dacli/worktrees/periodica-044-..., but dacli wait finalized run 01KZZTFZ6GH2RMA9PP41P5AG8A after 59s as 'no visible result' while the Codex process later completed and committed 5313f08. A resume invoked from the task worktree with dacli spawn --task 044 --role frontend-associate --runtime codex-terra-rw --detach --pr launched with cwd=/Users/tahabsn/Documents/GitHub/perio (dev), not the caller's task worktree, forcing the agent to restore its accidental edit and stop. dacli verify showed the same late-result race twice: it returned no-verdict, then transcripts recorded confirmed findings after exact-tree gates passed. Expected: preserve/resolve the task worktree cwd for resumed runs and wait for the runtime's final result/finding before finalizing detached or verify outcomes.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] A regression starts from an existing task worktree and spawns a follow-up agent for the same task without `--worktree`; the runtime cwd resolves to that task worktree, not the workspace main checkout.
- [x] If more than one candidate worktree or an unsafe mismatch exists, spawn refuses with exit 3 and names the explicit recovery command.
- [x] The resumed agent's edits and `dacli commit` land only on the task branch and preserve the original task/worktree attribution.
- [x] Spawns issued from the main checkout retain current behavior and do not inherit an unrelated task worktree.
- [x] Mutation evidence, focused execution/worktree tests, and `go test ./...` pass.
## Log
- 2026-08-16T17:07:36Z claimed by a-maintainer-gcwx9y
- 2026-08-16T17:23:57Z accepted by a-root
- 2026-08-16T17:23:57Z verified by `GOCACHE=/tmp/dacli-459-final go test ./...` (exit 0) in branch main at 65ff654 — proves that tree builds, not that the work is in trunk
- 2026-08-16T17:23:57Z deliverable: dacli/459-preserve-the-claimed-task-worktree-when-resuming-codex-agents is merged into main
- 2026-08-16T17:23:57Z completed by a-root
- 2026-08-16T17:30:15Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/675 (event 01M05S3EQTX5X1TXF349DXG0KM)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-459-accept go test ./...","exit_code":0,"duration_ms":73340,"artifact_hash":"sha256:58906d3550361ee7d9172a91acbd9e31db207c1714cb50e5bdd2517d476435f7","verifier":"a-root","branch":"main","commit_sha":"65ff6544cead81e9dde538661e5c02ee9b8eee57"}
{"command":"GOCACHE=/tmp/dacli-459-final go test ./...","exit_code":0,"duration_ms":70927,"artifact_hash":"sha256:b830415114f26a80396e93a8638abb9f566e0304cff86a449bf2b42b6ca4cf75","verifier":"a-root","branch":"main","commit_sha":"65ff6544cead81e9dde538661e5c02ee9b8eee57"}
