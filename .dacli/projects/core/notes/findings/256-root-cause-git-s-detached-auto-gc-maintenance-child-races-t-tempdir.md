---
id: f-256-root-cause-git-s-detached-auto-gc-maintenance-child-races-t-tempdir
kind: note
note_kind: finding
created: 2026-08-04T11:54:15Z
created_by: a-maintainer-g2363w
about: "[[256]]"
severity: moderate
---
# 256 root cause: git's detached auto-gc/maintenance child races t.TempDir RemoveAll
TestShipRecordMessageReportsActualMerges (and any shipEnv-based test) can fail in CI at t.TempDir cleanup with 'directory not empty' -- never at an assertion. Cause: git detaches auto gc/maintenance (gc.autoDetach defaults true) after a commit; the child keeps writing under .git after the git command and the test body return, so t.TempDir's deferred os.RemoveAll races it (ENOTEMPTY = a new entry appeared mid-remove). Local runs stay green because the trigger is CI's global git config enabling the auto path. Fix: shipEnv now sets repo-local gc.auto=0, gc.autoDetach=false, maintenance.auto=false (internal/features/ship/ship_test.go:42-52) which override any global config, so no detached git process is ever created. Guarded by TestShipEnvDisablesGitAutoMaintenance, which fails before the config lines exist. Verified: guard test fails without the fix (git config --get gc.auto -> exit 1 -> Fatal) and passes with it; full ship package green.
