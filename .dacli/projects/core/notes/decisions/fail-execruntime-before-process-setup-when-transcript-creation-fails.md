---
id: d-fail-execruntime-before-process-setup-when-transcript-creation-fails
kind: note
note_kind: decision
created: 2026-08-28T09:53:00Z
created_by: a-fixer-64tsev
about: "[[t-01M13S7VDH9ZN15AJAEYS5QFC4]]"
---
# Fail execRuntime before process setup when transcript creation fails
## Chose
Fail execRuntime before process setup when transcript creation fails
## Rejected
Continue the launch with a nil transcript sink
## Because
a reported runtime without its durable transcript makes wait, usage, recovery, and audit disagree with executed work
