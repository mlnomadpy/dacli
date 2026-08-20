---
id: f-global-installed-dacli-skill-is-stale-and-not-writable-from-the-task-worktree
kind: note
note_kind: finding
created: 2026-08-19T12:50:25Z
created_by: a-maintainer-6b3z6s
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: moderate
---
# Global installed dacli skill is stale and not writable from the task worktree
quick_validate.py reports repository skills/dacli valid, but diff -rq shows /Users/tahabsn/.codex/skills/dacli lacks the new focused references and differs from committed content. That global path is outside this task's writable roots; the publishable repository artifact is complete, but installation regeneration remains unverified here.
