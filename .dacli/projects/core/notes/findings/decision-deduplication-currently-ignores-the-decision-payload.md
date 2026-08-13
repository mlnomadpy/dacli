---
id: f-decision-deduplication-currently-ignores-the-decision-payload
kind: note
note_kind: finding
created: 2026-08-13T22:29:01Z
created_by: a-root
about: "[[437]]"
severity: major
---
# Decision deduplication currently ignores the decision payload
Owner review of commit a43daa7: canonicalNoteFiles groups every note kind solely by store.TitleSimilarity(title) >= 0.65. Two decisions with similar titles but materially different Chose/Rejected/Because sections would collapse, violating the requirement that distinct semantic records remain visible. Add a failing regression and make decision grouping compare normalized decision payload, while findings may continue treating bodies as retained evidence.
