---
id: f-075-complete-on-branch-dacli-075-accept-propose-survives-sync-r5-store-eventlog
kind: note
note_kind: finding
created: 2026-07-22T19:31:11Z
created_by: a-c7yhq42kbk
about: [[075]]
severity: moderate
---
# 075 complete on branch dacli/075...; accept-propose survives Sync; R5 store/eventlog defect closed
Commit f925ac5 by a-c7yhq42kbk on branch dacli/075-fix-store-accept-propose-survives-sync-store-eventlog-correctness. Staged ONLY: internal/eventlog/sync.go, internal/eventlog/sync_test.go, internal/features/acceptance/acceptance.go (--no-add). FIX: eventlog.apply's EventComment case now returns (false,"",nil) — leaving pending — when the body starts with the new eventlog.ProposePrefix ('accept-propose:'), so a Sync between an agent's propose and the owner's 'dacli accept' no longer consumes the proposal (sync.go:~168). The acceptance slice's proposePrefix is now = eventlog.ProposePrefix, giving the convention one definition (acceptance.go:45). Two new tests: TestSyncLeavesAcceptProposalPending (proposal stays pending + not logged) and TestSyncStillAppliesGenericComment (ordinary comments still log+apply, skip not over-matching). R5 SCOPING: the R5 audit finding (issue 34) verified all prior store/eventlog fixes HELD; the only open store/eventlog R5 defect was this accept-propose/sync race — now closed. The other open R5 defect (ghmirror seq substring, issue 32) lives in the ghmirror feature slice, NOT store/eventlog, so it is out of this task's scope. go build ./... clean; go test ./internal/... all green (env -u DACLI_AGENT to clear the leaked agent token).
