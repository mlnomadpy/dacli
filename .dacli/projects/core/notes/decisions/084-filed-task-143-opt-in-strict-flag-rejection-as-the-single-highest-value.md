---
id: d-084-filed-task-143-opt-in-strict-flag-rejection-as-the-single-highest-value
kind: note
note_kind: decision
created: 2026-07-24T09:27:31Z
created_by: a-vh51d10ng9
about: [[084]]
---
# 084: filed task 143 (opt-in strict-flag rejection) as the single highest-value evidence-based change
## Chose
084: filed task 143 (opt-in strict-flag rejection) as the single highest-value evidence-based change
## Rejected
A global unknown-flag reject inside clikit.ParseFlags, or filing yet another per-command flag-validation patch like the one init already got
## Because
ParseFlags is shared by every slice and Flags.Raw() exists specifically so 'run' can FORWARD unknown flags (clikit.go:153-155) -- a global reject would break pass-through commands. And a per-command hand-validation (what init got in wscore.go:30-45) does not fix the class: init STILL drops a typo'd --tempate silently. An opt-in Flags.Reject(known...) helper adopted on the agent-facing mutating commands is the minimal fix that closes the silent-drop class without touching pass-through semantics, matches the exit-2 usage contract, and is unit-testable.
