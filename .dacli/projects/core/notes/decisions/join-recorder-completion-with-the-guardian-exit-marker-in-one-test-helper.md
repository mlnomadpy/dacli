---
id: d-join-recorder-completion-with-the-guardian-exit-marker-in-one-test-helper
kind: note
note_kind: decision
created: 2026-08-26T13:41:18Z
created_by: a-fixer-5hgvyg
about: "[[t-01M0D4SN9N7MP3A02J76JZ32KW]]"
---
# Join recorder completion with the guardian exit marker in one test helper
## Chose
Join recorder completion with the guardian exit marker in one test helper
## Rejected
Treat the recorder completion marker as sufficient
## Because
the guardian writes runtime-exit.txt after its runtime child finishes, and that final write can race TempDir cleanup
