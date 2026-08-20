---
id: f-pushsync-replayed-stale-task-history-after-a-non-fast-forward-rejection
kind: note
note_kind: finding
created: 2026-08-19T14:12:38Z
created_by: a-maintainer-4243q0
about: "[[t-01M0D2KPGZZMYYSVSHNB8NS2T9]]"
severity: major
---
# PushSync replayed stale task history after a non-fast-forward rejection
internal/gitx/gitx.go:453 unconditionally rebased the checked-out branch onto origin/<task-branch>; the issue-726 fixture shows this changes HEAD and restores the obsolete remote topology after the local branch was rebased onto origin/main.
