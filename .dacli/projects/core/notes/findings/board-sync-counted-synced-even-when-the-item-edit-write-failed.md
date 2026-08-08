---
id: f-board-sync-counted-synced-even-when-the-item-edit-write-failed
kind: note
note_kind: finding
created: 2026-08-04T00:37:38Z
created_by: a-fixer-x41yjq
about: "[[205]]"
severity: moderate
---
# board sync counted synced++ even when the item-edit write failed
internal/features/ghmirror/project.go:441,465 (cmdProject's board loop) called setItemFields then unconditionally incremented synced, while setItemFields itself discarded every ghProjectCmd item-edit error via '_, _ ='. A rate-limited or failing write was reported to the operator as a successful field-sync. Fixed: setItemFields now returns bool ok (false if any attempted write errored), and the loop only increments synced when ok is true.
