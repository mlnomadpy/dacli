---
id: f-correction-three-additional-usage-drifts-remain-uncommitted
kind: note
note_kind: finding
created: 2026-08-19T12:30:02Z
created_by: a-root
about: "[[470]]"
severity: major
---
# Correction: three additional usage drifts remain uncommitted
The prior finding says next, shortcut add, and template show were aligned, but claim enforcement caused those unclaimed edits to be reverted before commit 9eb79cd. Their command tables still advertise queue next, queue add, and skill show. A follow-up with explicit claims must land those fixes and the command-path prefix invariant.
