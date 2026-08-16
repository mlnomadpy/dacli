---
id: d-require-agreement-in-problem-and-acceptance-overlap-for-content-based-duplicate
kind: note
note_kind: decision
created: 2026-08-16T17:45:45Z
created_by: a-maintainer-n5gm5y
about: "[[t-01KZZR4CN0HWN232ZD2GYGQDFP]]"
---
# Require agreement in problem and acceptance overlap for content-based duplicate refusal
## Chose
Require agreement in problem and acceptance overlap for content-based duplicate refusal
## Rejected
Treat aggregate title, context, and acceptance tokens as one similarity bag
## Because
Two generated-reference defects can share execution.go and generic command words; requiring both the observable problem and acceptance boundary to overlap catches the cycle-95 duplicate without refusing the shell-quoting contrast
