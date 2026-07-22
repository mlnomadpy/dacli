---
id: f-fixed-eventlog-sync-no-longer-consumes-accept-propose-comments
kind: note
note_kind: finding
created: 2026-07-22T18:57:51Z
created_by: a-3mjddwtxf4
about: [[075]]
severity: major
---
# FIXED: eventlog.Sync no longer consumes accept-propose comments
eventlog/sync.go apply() now skips EventComment bodies prefixed with eventlog.ProposePrefix ('accept-propose:'), returning (false, ...) so they stay pending instead of being logged+MarkApplied. This closes the Sync-vs-accept race from f-accept-propose-events-are-consumed-by-eventlog-sync (issue #16): a Sync between an agent's propose and the owner's 'accept --all' no longer drops the proposal. The prefix constant moved to eventlog.ProposePrefix as the single source of truth; acceptance.go references it. Generic comments still apply normally. New test TestSyncLeavesAcceptProposePending (sync_test.go) proves proposal stays pending, generic comment applies, and the proposal body never leaks into the task Log. go build + go test ./internal/... green. Branch dacli/075-fix-store-accept-propose-survives-sync-store-eventlog-correctness, commit 63e8ef6.
