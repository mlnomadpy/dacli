---
id: 01M146TW4VBJPBZQZ17D98R4X4
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-28T12:52:06Z
created_by: a-fixer-fzgpye
about: "[[t-01M11HZW8X678270CFWDBK7YTN]]"
origin: agent
applied: true
checksum: sha256:710931fcf065fed0070cf066fdaa74601d747c22b4e81c4e11c7af0c57ee6f14
---
5ed712d8 t-01M11HZW8X678270CFWDBK7YTN: prefer an active task's open PR over historical landing

An active task can intentionally land multiple PR slices from its canonical branch. Resolve its current open PR before treating an older integration event as final, so integrate processes later work instead of cleaning it up. Regression mutation: removing the active-task open-PR preference makes TestActiveTaskPrefersCurrentOpenPROverHistoricalMerge fail with status merged for pull/700 rather than the current pull/701.
role: fixer
