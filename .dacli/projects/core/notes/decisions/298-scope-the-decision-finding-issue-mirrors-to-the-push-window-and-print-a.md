---
id: d-298-scope-the-decision-finding-issue-mirrors-to-the-push-window-and-print-a
kind: note
note_kind: decision
created: 2026-08-04T20:37:57Z
created_by: a-maintainer-qtd48g
about: "[[298]]"
---
# 298: scope the decision+finding-issue mirrors to the push window and print a blast-radius plan line, rather than refusing windowed pushes
## Chose
298: scope the decision+finding-issue mirrors to the push window and print a blast-radius plan line, rather than refusing windowed pushes
## Rejected
Refuse any push that combines a task window with the decision/finding mirrors
## Because
decisions currently ALWAYS ride every push (mirrorDecisions runs unconditionally), so a blanket refusal-on-window would make task 275's window feature unusable whenever any decision exists. Scoping via a shared noteInWindow predicate (about-match on the ref-selected tasks OR created>=since, mirroring selectTaskWindow's two axes) publishes exactly what was named — 'a scoped push publishes exactly what was named' — while keeping the default whole-project mirror unchanged.
