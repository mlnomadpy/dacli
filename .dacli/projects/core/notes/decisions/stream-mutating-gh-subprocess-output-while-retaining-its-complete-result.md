---
id: d-stream-mutating-gh-subprocess-output-while-retaining-its-complete-result
kind: note
note_kind: decision
created: 2026-08-28T00:41:21Z
created_by: a-fixer-0mrzjc
about: "[[t-01M1068MTFPQ6YFVQG204M2EX4]]"
---
# Stream mutating gh subprocess output while retaining its complete result
## Chose
Stream mutating gh subprocess output while retaining its complete result
## Rejected
Buffer every gh subprocess until exit
## Because
The public github push must expose active publication progress without changing read-probe output or terminal error parsing.
