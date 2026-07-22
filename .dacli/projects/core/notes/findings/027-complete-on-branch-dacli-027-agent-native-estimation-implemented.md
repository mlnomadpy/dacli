---
id: f-027-complete-on-branch-dacli-027-agent-native-estimation-implemented
kind: note
note_kind: finding
created: 2026-07-22T13:29:06Z
created_by: a-sr9e6xf1d0
about: [[027]]
severity: moderate
---
# 027 complete on branch dacli/027-...: agent-native estimation implemented
Commit 440962b by a-sr9e6xf1d0. Staged ONLY 3 files (git add + dacli commit --no-add): internal/store/calibration.go, internal/features/insight/insight.go, internal/features/execution/execution.go. (1) execution.go:291 invocation.txt now writes 'model: <modelName>' (clikit.OrDash) so a run record carries role×model×runtime — the ONLY execution.go change. (2) calibration.go: new Band{Role,Model,Runtime} type; CalibrationSamples now joins each done task to its completing run via runBands(w) which scans every RunsDir/<run>/invocation.txt for task/role/model/runtime (last-matching run wins, ULID order = chronological); CalibSample gains Band; new TaskBand(w,taskID) helper. (3) insight.go cmdCalibrate keeps the size-band view and adds a 'by agent band (role/model/runtime)' section printing n, median, and p10–p90 spread (new local percentile() helper, linear interp — spm is out of scope so no spm change); bands with n>=10 marked AUTHORITATIVE. cmdEstimate surfaces the empirical band distribution AS the estimate (×ratio median + p10–p90, projected to hours via Te) with PERT labelled the prior, when the task's own run band has n>=10. go build ./... clean; go test ./internal/... all green (incl. store, cli with TestMain clearing DACLI_AGENT). NOTE: read-only smoke run of 'dacli calibrate' was blocked by the headless sandbox (binary exec needs approval), so the readout was verified via build+unit tests only. Owner: verify and close via task check/done + dacli merge --task 027.
