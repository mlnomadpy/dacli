---
id: d-infer-task-381-scope-from-explicit-architectural-and-command-vocabulary
kind: note
note_kind: decision
created: 2026-08-12T19:04:31Z
created_by: a-codex-maintainer-cr0hke
about: "[[393]]"
github:
  issue: 522
  repo: mlnomadpy/dacli
---
# Infer task 381 scope from explicit architectural and command vocabulary
## Chose
Infer task 381 scope from explicit architectural and command vocabulary
## Rejected
infer from generic creation, resizing, and validation words
## Because
spm.ThreePoint, task add, task estimate, and critical-path map directly to stable package boundaries while generic behavioral terms would overclaim unrelated feature slices
