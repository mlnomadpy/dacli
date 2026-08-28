---
id: f-task-511-verified-patch-is-blocked-by-worktree-ownership-mismatch
kind: note
note_kind: finding
created: 2026-08-28T09:10:01Z
created_by: a-maintainer-fb7kb1
about: "[[t-01M1068MTFPQ6YFVQG204M2EX4]]"
severity: major
---
# task 511 verified patch is blocked by worktree ownership mismatch
In .dacli/worktrees/core-511-agent-report-github-push-returns-before-the-publisher-releases-its-sequence-lock, /tmp/dacli-main commit as DACLI_AGENT a-maintainer-fb7kb1 refused because the worktree is owned by a-root. The command says staged work was preserved and requires root to preview dacli worktree reclaim --claim for the five changed ghmirror/CLI test paths. Raw git commit was not used because it would lose attribution.
