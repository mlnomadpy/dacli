---
id: f-016-complete-on-branch-dacli-016-resource-monitoring-and-kill-for-spawned-agent-process-trees
kind: note
note_kind: finding
created: 2026-07-22T10:02:14Z
created_by: a-zqtmzbe6mv
about: [[016]]
severity: moderate
---
# 016 complete on branch dacli/016-resource-monitoring-and-kill-for-spawned-agent-process-trees
Commit 2129a51 by a-zqtmzbe6mv (maintainer). Staged ONLY: internal/procmon/{procmon.go,procmon_test.go}, internal/features/execution/{execution.go,verify.go}, .dacli/roles/maintainer.md, and the 016 task file (used --no-add so the unrelated parallel-run working-tree state was excluded). All 4 acceptance criteria are satisfied: (1) execRuntime sets SysProcAttr.Setpgid and cmd.Cancel SIGKILLs the whole -pgid group on timeout; (2) dacli agents lists live-probed agents with group RAM/CPU/procs/uptime and GPU honestly n/a when nvidia-smi absent; (3) dacli kill <ref|--all> runs KillTree SIGTERM->SIGKILL-after-grace on the whole group and writes killed.txt audit crumb; (4) committed on branch, go build clean + go test green. Box-checking is refused for non-owners (only a-root checks boxes) so the owner should verify/close via dacli task check/done + dacli merge --task 016.
