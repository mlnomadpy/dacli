---
id: d-gradefinding-prefers-id-match-then-grades-all-title-twins
kind: note
note_kind: decision
created: 2026-07-22T16:25:54Z
created_by: a-n2q5ysnx5y
about: [[048]]
---
# GradeFinding prefers id-match, then grades ALL title twins
## Chose
GradeFinding prefers id-match, then grades ALL title twins
## Rejected
silently pick the first os.ReadDir hit, or refuse on any title ambiguity
## Because
An exact id is unique so it targets the intended note deterministically; when only the claim text (title) matches and two notes share it (CreateNote's own collision path expects same-titled twins), the verify verdict is ABOUT that claim, so every note asserting it earns the same grade — grading all twins is deterministic (order-independent) and makes progress, whereas refusing leaves real findings ungraded and picking-first is the nondeterminism the fix removes.
