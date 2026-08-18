---
id: f-loop-checkpoints-discarded-every-recovery-ledger-persistence-failure
kind: note
note_kind: finding
created: 2026-08-18T14:41:19Z
created_by: a-maintainer-dgyp5f
about: "[[t-01M0AEG5F23TRH6BAR9HT38ZP1]]"
severity: major
---
# Loop checkpoints discarded every recovery-ledger persistence failure
internal/features/orchestration/journal.go writeCycleJournal ignored directory creation, atomic replacement, and empty-ledger removal errors; internal/features/orchestration/orchestration.go saveState then reported checkpoints after losing pending accepts/lands, the token ceiling, and landing policy.
