---
id: 01KYGA8JMKJRWQNAJTZNR399SH
kind: event
event_kind: finding
created: 2026-07-26T22:55:49Z
created_by: a-vav46gnkax
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: true
---
loop --pr pendingAccept/pendingLand not persisted: bounded/restarted loop never closes merged tasks and re-opens duplicate PRs

The 115 fix defers 'accept --force' until PR-merge is confirmed by parking each built task in d.pendingAccept (orchestration.go:256,479) and holding its push via d.pendingLand (orchestration.go:255,480). BUT both slices live ONLY on the in-memory driver struct, constructed fresh at orchestration.go:152 with no restore — while the governor's counters ARE persisted+restored (orchestration.go:132-133 via state.go writeGovernorState/readGovernorState). reconcilePendingAccepts (orchestration.go:558) and excludePending (orchestration.go:641) both short-circuit on len(pendingAccept)==0. REPRO: 'dacli loop --pr --max-cycles 1' (the bounded build-itself sprint pattern) builds task 200, opens a self-PR with auto-merge queued, parks {200,branch} in pendingAccept, exits at MaxCycles — pendingAccept discarded. GitHub merges the PR minutes later on green CI. Next bounded invocation: new driver, pendingAccept=nil -> reconcilePendingAccepts returns at line 558, so task 200 is NEVER accepted despite its PR merging (false-open — the inverse of the 115 bug), AND excludePending(ready,nil) returns ready unchanged (line 641) so task 200 re-enters the frontier and the loop rebuilds it, opening a SECOND PR on already-merged work (duplicate spawn/tokens/branch). Under the dominant bounded/perpetual-with-restart usage the entire 115 guarantee evaporates one process boundary later. Same self-regressing 'loop state not persisted across restarts' class as governor persistence (task 096), reintroduced by 115 adding new loop state without extending state.go. Recoverable: prLandStatus already classifies any branch from the workspace alone; reconcile just needs to scan open --pr tasks with live branches instead of only the in-memory queue (or persist pendingAccept alongside governorState).
