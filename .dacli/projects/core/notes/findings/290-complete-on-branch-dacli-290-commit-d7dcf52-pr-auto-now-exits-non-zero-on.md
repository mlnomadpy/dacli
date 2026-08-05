---
id: f-290-complete-on-branch-dacli-290-commit-d7dcf52-pr-auto-now-exits-non-zero-on
kind: note
note_kind: finding
created: 2026-08-04T20:20:55Z
created_by: a-maintainer-6qp5vh
about: "[[290]]"
severity: major
---
# 290 complete on branch dacli/290-... (commit d7dcf52): pr --auto now exits non-zero on auto-merge queue failure; loop distinguishes stranded from landing
Both acceptance criteria met, build+vet+full go test green, gofmt clean, verified by reproduction.

CRITERION 1 (failure visible in exit status, not only stderr): lifecycle.go cmdPR --auto previously printed 'note: auto-merge not queued' to stderr and returned nil (EXIT 0). Now extracted into queueAutoMerge(root,branch) (lifecycle.go:311) which RETURNS the failure as an error, so cmdPR returns non-zero (exit 1) — a headless caller reading exit code no longer believes a stranded PR landed. Network-unreachable gets its own message vs 'Allow auto-merge off'. This matches the sibling integrate --pr --auto (prIntegrateTask lifecycle.go:1239-1249) which already treated the identical failure as fatal — the two --auto surfaces now agree (closes finding f-land-dacli-pr-auto-exits-0-with-only-a-stderr-note).

CRITERION 2 (loop distinguishes queued from not-queued): orchestration.go prLandStatus now fetches autoMergeRequest and returns a new 'stranded' state for an OPEN PR with NO auto-merge queued (vs 'landing' when queued). reconcilePendingAccepts logs a stranded PR loudly ('PR open but NOT queued for auto-merge — it will NOT self-land') instead of silently parking it as if it were landing. Kept pending (not dropped) so the loop keeps watching without opening a duplicate PR.

TESTS (all fail before the change, verified by neutering each fix and re-running): vcs TestQueueAutoMergeFailureIsFatal (unit, no gh needed), vcs TestPRAutoStrandedExitsNonZero (end-to-end cmdPR, gh-guarded), orchestration TestReconcilePendingAcceptsFlagsStrandedPR. Existing TestReconcilePendingAcceptsKeepsWaitingWhilePROpen updated to carry autoMergeRequest so it exercises the true landing path.

DOCS: internal/prompts/tpl/git_workflow.md + testdata updated — removed the now-false 'always safe to pass / degrades to leaving the PR open' claim, replaced with the fail-loud behavior.

PR-first is off for this run: owner runs dacli accept 290 then integrate/merge --task 290.
