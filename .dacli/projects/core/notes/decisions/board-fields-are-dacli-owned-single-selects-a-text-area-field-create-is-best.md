---
id: d-board-fields-are-dacli-owned-single-selects-a-text-area-field-create-is-best
kind: note
note_kind: decision
created: 2026-07-22T20:15:07Z
created_by: a-rncv68m0fd
about: [[071]]
---
# board fields are dacli-owned single-selects + a text Area; field-create is best-effort with reuse-by-name
## Chose
board fields are dacli-owned single-selects + a text Area; field-create is best-effort with reuse-by-name
## Rejected
reuse the built-in Projects-v2 Status field and remap our four folders onto its Todo/In-Progress/Done options
## Because
gh project cannot add options to an existing single-select, so remapping onto the built-in Status (Todo/In-Progress/Done) could not represent open/active/blocked/done; instead ensureFields reuses any field by name (adopting a built-in Status when present) and otherwise creates our own SINGLE_SELECT with the exact option set, while setItemFields resolves a value NAME to an option id and SKIPS a value that is not an option (optionID=='') so a board carrying an incompatible built-in Status leaves that field unset rather than failing the whole sync — Area is TEXT because area slices are dynamic and unbounded
