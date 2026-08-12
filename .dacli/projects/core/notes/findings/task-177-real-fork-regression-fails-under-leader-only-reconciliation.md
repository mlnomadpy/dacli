---
id: f-task-177-real-fork-regression-fails-under-leader-only-reconciliation
kind: note
note_kind: finding
created: 2026-08-12T14:14:35Z
created_by: a-codex-maintainer-1ecns6
about: "[[375]]"
severity: major
---
# Task 177 real fork regression fails under leader-only reconciliation
internal/features/execution/runstilllive_unix_test.go starts a Setpgid shell that forks sleep, records proc.txt, then exits. Before the fix, GOCACHE=/private/tmp/dacli-375-gocache go test ./internal/features/execution -run 'TestRunStillLivePreservesTask(177|369)' -count=1 -v fails at runstilllive_unix_test.go with: task 177: reconciliation lost a genuine helper after its recorded leader exited; the task 369 recycled-group case passes.
