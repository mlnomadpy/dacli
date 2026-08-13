---
id: f-correction-task-430-changes-are-preserved-but-unstaged-after-commit-refusal
kind: note
note_kind: finding
created: 2026-08-13T20:08:18Z
created_by: a-codex-maintainer-zkfgn1
about: "[[430]]"
severity: moderate
---
# Correction task 430 changes are preserved but unstaged after commit refusal
Supersedes the staged wording in the prior finding: git status --short shows a blank index column and M in the worktree column for all four internal/eventlog files. The files are preserved and verified but not staged, despite dacli's refusal message saying 'staged work was preserved'. dacli report could not file upstream because the configured gh token is invalid.
