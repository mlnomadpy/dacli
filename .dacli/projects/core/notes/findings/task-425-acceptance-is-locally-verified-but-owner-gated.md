---
id: f-task-425-acceptance-is-locally-verified-but-owner-gated
kind: note
note_kind: finding
created: 2026-08-13T16:15:45Z
created_by: a-codex-maintainer-sg0bxk
about: "[[425]]"
severity: major
---
# Task 425 acceptance is locally verified but owner-gated
All four criteria are observed: restart retains signed outbox envelopes plus inbox cursor; retry preserves ids and signatures across delayed, duplicate, reordered, replayed, incompatible, and tampered fixtures; invalid signatures and versions leave state.json and inbox unchanged; focused and full Go tests pass. task check 425 --n 1 returned policy refusal because only owner a-root may check boxes, so no other boxes were attempted.
