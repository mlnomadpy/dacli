---
id: d-persist-verifier-required-recovery-state-on-pending-accept-entries
kind: note
note_kind: decision
created: 2026-08-16T17:25:41Z
created_by: a-maintainer-psevtg
about: "[[t-01KZYW7M979TQNHD2VTA1Q9WAT]]"
---
# Persist verifier-required recovery state on pending accept entries
## Chose
Persist verifier-required recovery state on pending accept entries
## Rejected
Retry accept --force each cycle or drop every failed merged entry
## Because
Exit 3 must not be retried, while dropping an open task loses the only recovery ledger; an optional journal token preserves old two-field entries and suppresses repeated command execution and messaging.
