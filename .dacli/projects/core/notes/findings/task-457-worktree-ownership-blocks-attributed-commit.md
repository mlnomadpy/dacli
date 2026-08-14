---
id: f-task-457-worktree-ownership-blocks-attributed-commit
kind: note
note_kind: finding
created: 2026-08-14T09:35:07Z
created_by: a-maintainer-rmzh0s
about: "[[t-01KZZSD1K4YT88J0YYB5ZPD75R]]"
severity: major
---
# Task 457 worktree ownership blocks attributed commit
After implementation and verification, dacli commit refused with exit 3: worktree is owned by a-maintainer-2260xw but this run was issued DACLI_AGENT for a-maintainer-rmzh0s. The command says staged work was preserved. Recovery requires the original a-maintainer-2260xw token; raw git commit was not used.
