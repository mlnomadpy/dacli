---
id: f-cmdask-claims-the-owner-s-dacli-sync-applies-it-but-eventlog-sync-never-applies
kind: note
note_kind: finding
created: 2026-08-06T08:23:59Z
created_by: a-fixer-43t0j0
about: "[[299]]"
severity: minor
---
# cmdAsk claims 'the owner's dacli sync applies it' but eventlog.Sync never applies EventHelp
internal/features/collab/collab.go:118-119 (cmdAsk) tells a read-only caller: '<agent> cannot block a task it does not own, so the owner's `dacli sync` applies it.' But internal/eventlog/sync.go's apply() switch (sync.go:96-244) has no case for model.EventHelp — it falls to the default at sync.go:240-243, which explicitly says 'help/answer/run materialize in later wedges; leaving them pending is honest.' So running `dacli sync` after a read-only agent's `dacli ask` does NOT move the task to blocked; only `dacli status`/doctor read the pending EventHelp directly (insight.go:1089) for reporting. Found while wiring the loop's new per-cycle 'dacli sync' call (task 299) and confirming which read-only-agent proposals it actually promotes — EventClaim/Release/Finding/ProposeStatus/Comment/Block are all applied; EventHelp is not, despite the user-facing message implying it is. Not fixed here: out of 299's acceptance scope, and the right fix (either implement EventHelp in Sync, or correct the message) needs its own decision about which is intended.
