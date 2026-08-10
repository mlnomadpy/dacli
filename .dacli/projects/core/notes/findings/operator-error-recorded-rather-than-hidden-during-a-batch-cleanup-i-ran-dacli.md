---
id: f-operator-error-recorded-rather-than-hidden-during-a-batch-cleanup-i-ran-dacli
kind: note
note_kind: finding
created: 2026-08-10T18:51:18Z
created_by: a-root
origin: .dacli/projects/core/tasks/open/336-github-push-creates-decision-issues-but-never-closes-them-so-records-accumulate.md:1
---
# Operator error, recorded rather than hidden: during a batch cleanup I ran 'dacli accept --force' against tasks 336 and 339 without checking what they were. 336 (github push never closes decision issues) had NONE of its work done — I closed those issues by hand on GitHub, which is not the same thing — and its three acceptance boxes were checked by the force-accept. 339 was a throwaway routing probe that should never have been a task; an earlier 'dacli task rm' printed help instead of removing it and I did not check. Both were reopened/removed by editing the store directly, because there is no 'task reopen' or 'task rm' command: a wrongly-closed task can only be corrected by hand. Two things the tool did right and I ignored: accept stamped 'closed WITHOUT verification' on both, and 'task rm' does not exist so nothing silently deleted anything. The lesson is the one this session keeps relearning in the other direction — --force exists to reconcile orphaned owners, not to skip reading what you are closing.
