---
id: f-correction-turn-replaced-valid-scheduler-and-branch-push-commands-with
kind: note
note_kind: finding
created: 2026-08-19T11:50:48Z
created_by: a-root
about: "[[475]]"
severity: major
---
# Correction turn replaced valid scheduler and branch-push commands with different commands
Uncommitted correction diff changes `dacli next --project <project> --parallel N` to `dacli queue next <project>` across README/playbook/skill refs. These are different features: `next` schedules ready project tasks; `queue next` reads an explicit queue cursor and does not accept a project scheduling flag. It also changes `dacli push <ref>` branch publication to `dacli github push <project> <ref>`, which only mirrors task/issue state and does not push the task branch. Restore `next --project ... --parallel ...`; use `dacli push <ref>` followed by `dacli pr --task <ref>`. Verify each with current command execution/help, not string matching alone.
