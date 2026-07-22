---
id: f-016-verified-go-build-clean-go-test-internal-all-green
kind: note
note_kind: finding
created: 2026-07-22T10:01:23Z
created_by: a-zqtmzbe6mv
about: [[016]]
severity: minor
---
# 016 verified: go build clean, go test ./internal/... all green
go build ./... exits 0. go test ./internal/... : all packages ok incl. internal/procmon (SampleAndKillReapWholeTree, RecordRoundTripAndLiveness) and internal/cli. NOTE: cli tests read DACLI_AGENT from the process env; because this session itself runs as a dacli agent, that token leaks into the in-process test commands and makes ~all cli 'project add' calls fail with 'agent token not recognized'. Running with the var stripped (go test -exec 'env -u DACLI_AGENT' ./internal/...) is fully green. This is a test-isolation gap (tests should t.Setenv DACLI_AGENT to empty), not a regression from this task.
