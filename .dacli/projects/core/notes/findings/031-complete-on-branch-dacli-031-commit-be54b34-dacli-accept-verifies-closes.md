---
id: f-031-complete-on-branch-dacli-031-commit-be54b34-dacli-accept-verifies-closes
kind: note
note_kind: finding
created: 2026-07-22T14:51:17Z
created_by: a-yqfsr7b052
about: [[031]]
severity: major
---
# 031 complete on branch dacli/031 — commit be54b34, dacli accept verifies+closes, agents propose via events
Committed be54b34 by a-yqfsr7b052 (maintainer) via git add + dacli commit --no-add, staging ONLY the 3 scoped files: internal/features/acceptance/acceptance.go (new slice), internal/store/store.go (CheckAllAcceptance helper), internal/cli/cli.go (register slice). ACCEPTANCE, both criteria satisfied and verified end-to-end via a throwaway internal/cli test (deleted, not staged): (1) dacli accept <ref> [--verify "cmd"] runs the optional verification hook — a non-zero exit REFUSES the accept (exit 1, task stays open, no boxes checked) — then store.CheckAllAcceptance checks every acceptance box, SaveTask + MoveTask(done) closes it in one owner step. --all accepts every task an agent has proposed, gating the whole batch once with --verify. Owner-only via id.CanMutate(t.Owner()), mirroring task check/done. (2) PROPOSE path: a read-only agent running dacli accept <ref> records an EventComment with body prefix 'accept-propose:' (a comment, not a finding — an intention, not a fact; new event kinds are out of 031 scope); the owner's dacli accept / accept --all applies the box-checks and MarkApplied's the event (idempotent — a re-run finds nothing). Slice imports NO other feature slice (agentid/clikit/eventlog/model/store/workspace only), so TestFeatureSlicesAreIsolated stays green. go build ./... clean; go test ./internal/... all green incl. internal/cli and internal/store. Box-checking is owner-only — owner should verify and close via dacli accept 031 (dogfood) or dacli task check/done, then dacli merge --task 031.
