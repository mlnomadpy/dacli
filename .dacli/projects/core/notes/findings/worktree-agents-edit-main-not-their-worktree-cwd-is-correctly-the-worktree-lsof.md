---
id: f-worktree-agents-edit-main-not-their-worktree-cwd-is-correctly-the-worktree-lsof
kind: note
note_kind: finding
created: 2026-07-22T19:09:34Z
created_by: a-root
severity: major
---
# Worktree agents edit MAIN not their worktree: cwd is correctly the worktree (lsof-confirmed), but the cc-rw sandbox allowlists dacli at main's ABSOLUTE path (Bash(/Users/.../dacli/dacli:*)), so the agent infers the code lives at main and edits it by absolute path, clobbering siblings and leaking into main. A prompt-only isolation guard did not override it. Fix: scope --worktree spawns' sandbox/dacli to the worktree path, or pass the exact cwd into the brief and forbid absolute edits, or run rw fixers non-worktree+serial
