---
id: f-021-complete-on-branch-dacli-021-async-spawn-wait-agent-lifecycle-features-from
kind: note
note_kind: finding
created: 2026-07-22T12:06:14Z
created_by: a-yyn9jj4j0b
about: [[021]]
severity: moderate
---
# 021 complete on branch dacli/021-async-spawn-wait-agent-lifecycle-features-from-my-dacli-feedback
Committed 7df2659 by a-yyn9jj4j0b (maintainer) on branch dacli/021-async-spawn-wait-agent-lifecycle-features-from-my-dacli-feedback. Staged ONLY the 9 intended files (git add + dacli commit --no-add, so unrelated .dacli/** workspace churn was excluded): internal/features/execution/{execution.go,verify.go}, internal/features/vcs/vcs.go, internal/procmon/procmon.go, internal/prompts/tpl/protocol_preamble.md, internal/store/store.go, internal/spm/criticalpath.go, internal/cli/main_test.go, and the 021 task file. All 5 acceptance criteria satisfied and verified in the diff: (1) spawn --detach returns run-id immediately + dacli wait blocks & finalizeRun derives outcome from workspace effects; (2) worktree spawn copies child agent-file into worktree (self-recognition, issue #1) + outcome reads eventsWS; (3) headless preamble forbids waiting + cli TestMain clears DACLI_AGENT + commit warns on unresolved role + Slugify capped to 80; (4) logs -f follows transcript + spawn --claim PathsOverlap refusal + agents --max-rss/--max-runtime --reap; (5) committed on branch, go build clean + go test ./internal/... green. Box-checking refused for non-owners (only a-root) — owner should verify and close via dacli task check/done + dacli integrate/merge --task 021.
